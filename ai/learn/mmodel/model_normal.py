"""
 - Модель ImprovedChessNet (ResBlocks + опциональный Transformer), имеет две головы:
    - policy (классификация по словарю ходов)
    - value (оценка результата партии)
 - Использует mixed precision (AMP), OneCycleLR, сохранение лучшей модели
"""


from typing import Dict, Optional
import pandas as pd
import os
import json
import numpy as np
import chess
import torch
import torch.nn as nn
import torch.nn.functional as F
from torch.utils.data import Dataset
from tqdm import tqdm
from mtools import mtools

# --------------------------- Dataset ---------------------------
class ChessDataset(Dataset):
    def __init__(self, df: pd.DataFrame, move2idx: Dict[str,int], max_samples: Optional[int]=0):
        self.df = df.reset_index(drop=True)
        if max_samples != 0 and max_samples < len(self.df):
            self.df = self.df.sample(n=max_samples, random_state=42).reset_index(drop=True)
        self.move2idx = move2idx

    def __len__(self):
        return len(self.df)

    def __getitem__(self, idx):
        row = self.df.iloc[idx]
        fen = row['fen']
        move = str(row['move'])
        rating = float(row['rating']) if 'rating' in row and not pd.isna(row['rating']) else 2000.0
        result = float(row['result']) if 'result' in row and not pd.isna(row['result']) else 0.5

        x = mtools.fen_to_tensor(fen)  # (13,8,8)
        # policy target
        y = self.move2idx.get(move, None)
        if y is None:
            # If unseen move (should be rare if vocab built from full dataset), map to a fallback index 0
            y = 0
        # value target: scale to [-1,1]
        v = (result * 2.0) - 1.0
        # normalize rating (optional) -> we'll scale to ~0..1 by dividing by 3000
        rating_norm = rating / 3000.0
        return torch.tensor(x, dtype=torch.float32), torch.tensor(y, dtype=torch.long), torch.tensor(v, dtype=torch.float32), torch.tensor(rating_norm, dtype=torch.float32)


def collate_fn(batch):
    xs = torch.stack([item[0] for item in batch])
    ys = torch.stack([item[1] for item in batch])
    vs = torch.stack([item[2] for item in batch])
    ratings = torch.stack([item[3] for item in batch]).unsqueeze(1)
    return xs, ys, vs, ratings


# --------------------------- Model ---------------------------
class ResBlock(nn.Module):
    def __init__(self, channels):
        super().__init__()
        self.conv1 = nn.Conv2d(channels, channels, kernel_size=3, padding=1, bias=False)
        self.bn1 = nn.BatchNorm2d(channels)
        self.conv2 = nn.Conv2d(channels, channels, kernel_size=3, padding=1, bias=False)
        self.bn2 = nn.BatchNorm2d(channels)
        self.relu = nn.ReLU(inplace=True)

    def forward(self, x):
        identity = x
        out = self.conv1(x)
        out = self.bn1(out)
        out = self.relu(out)
        out = self.conv2(out)
        out = self.bn2(out)
        out += identity
        out = self.relu(out)
        return out

class TinyTransformer(nn.Module):
    def __init__(self, dim, nhead=4, nlayers=2, dim_feedforward=512, dropout=0.1):
        super().__init__()
        encoder_layer = nn.TransformerEncoderLayer(d_model=dim, nhead=nhead,
                                                   dim_feedforward=dim_feedforward,
                                                   dropout=dropout, activation='relu',
                                                   batch_first=True)
        self.encoder = nn.TransformerEncoder(encoder_layer, num_layers=nlayers)
        self.pos_emb = nn.Parameter(torch.randn(64, dim))

    def forward(self, x):
        B, C, H, W = x.shape
        t = x.view(B, C, H*W).permute(0, 2, 1)  # (B, 64, C)
        t = t + self.pos_emb.unsqueeze(0)
        t = t.permute(1, 0, 2)  # (S, B, E)
        t = self.encoder(t)
        t = t.permute(1, 0, 2)  # (B, S, E)
        pooled = t.mean(dim=1)
        return pooled

class ImprovedChessNet(nn.Module):
    def __init__(self, num_moves:int, use_transformer:bool=True, scalar_feat_dim:int=1):
        super().__init__()
        self.stem = nn.Sequential(
            nn.Conv2d(13, 64, kernel_size=3, padding=1, bias=False),
            nn.BatchNorm2d(64),
            nn.ReLU(inplace=True)
        )
        self.res_blocks = nn.Sequential(
            ResBlock(64),
            ResBlock(64),
            ResBlock(64),
        )
        self.project = nn.Conv2d(64, 128, kernel_size=1)
        self.use_transformer = use_transformer
        if use_transformer:
            self.transformer = TinyTransformer(dim=128, nhead=8, nlayers=2, dim_feedforward=512, dropout=0.1)
            hidden_dim = 128
        else:
            hidden_dim = 128

        self.scalar_proj = nn.Sequential(
            nn.Linear(scalar_feat_dim, 32),
            nn.ReLU(inplace=True),
            nn.Linear(32, 32),
            nn.ReLU(inplace=True)
        )

        self.fc = nn.Sequential(
            nn.Linear(hidden_dim + 32, 512),
            nn.ReLU(inplace=True),
            nn.Dropout(0.3),
        )

        self.policy_head = nn.Linear(512, num_moves)
        self.value_head = nn.Sequential(
            nn.Linear(512, 128),
            nn.ReLU(inplace=True),
            nn.Linear(128, 1),
            nn.Tanh()
        )

    def forward(self, x, scalar_feats=None):
        b = x.size(0)
        out = self.stem(x)
        out = self.res_blocks(out)
        out = self.project(out)
        if self.use_transformer:
            pooled = self.transformer(out)
        else:
            pooled = F.adaptive_avg_pool2d(out, (1,1)).view(b, -1)

        if scalar_feats is None:
            sf = torch.zeros(b, 1, device=out.device)
        else:
            sf = scalar_feats
        sf_proj = self.scalar_proj(sf)
        merged = torch.cat([pooled, sf_proj], dim=1)
        core = self.fc(merged)
        policy_logits = self.policy_head(core)
        value = self.value_head(core)
        return policy_logits, value.squeeze(1)


# --------------------------- Training / Eval ---------------------------
def build_move_vocab(moves):
    uniq = sorted(list(set(moves)))
    move2idx = {m:i for i,m in enumerate(uniq)}
    idx2move = {i:m for m,i in move2idx.items()}
    return move2idx, idx2move


def train_one_epoch(model, loader, opt, scaler, loss_fn_policy, alpha_value, device, scheduler=None):
    model.train()
    running_loss = 0.0
    running_policy_loss = 0.0
    running_value_loss = 0.0
    correct = 0
    total = 0

    for x,y,v,r in tqdm(loader, desc='train', leave=False):
        x = x.to(device)
        y = y.to(device)
        v = v.to(device)
        r = r.to(device)

        opt.zero_grad()
        with torch.cuda.amp.autocast(enabled=(device.type=='cuda')):
            logits, value_pred = model(x, scalar_feats=r)
            policy_loss = loss_fn_policy(logits, y)
            value_loss = F.mse_loss(value_pred, v)
            loss = policy_loss + alpha_value * value_loss
        scaler.scale(loss).backward()
        scaler.unscale_(opt)
        torch.nn.utils.clip_grad_norm_(model.parameters(), 1.0)
        scaler.step(opt)
        scaler.update()
        if scheduler is not None:
            try:
                scheduler.step()
            except Exception:
                pass

        running_loss += float(loss.item()) * x.size(0)
        running_policy_loss += float(policy_loss.item()) * x.size(0)
        running_value_loss += float(value_loss.item()) * x.size(0)
        preds = logits.argmax(dim=1)
        correct += (preds == y).sum().item()
        total += x.size(0)

    return running_loss/total, running_policy_loss/total, running_value_loss/total, correct/total


def eval_model(model, loader, loss_fn_policy, alpha_value, device):
    model.eval()
    running_loss = 0.0
    running_policy_loss = 0.0
    running_value_loss = 0.0
    correct = 0
    total = 0
    with torch.no_grad():
        for x,y,v,r in tqdm(loader, desc='eval', leave=False):
            x = x.to(device)
            y = y.to(device)
            v = v.to(device)
            r = r.to(device)
            logits, value_pred = model(x, scalar_feats=r)
            policy_loss = loss_fn_policy(logits, y)
            value_loss = F.mse_loss(value_pred, v)
            loss = policy_loss + alpha_value * value_loss
            running_loss += float(loss.item()) * x.size(0)
            running_policy_loss += float(policy_loss.item()) * x.size(0)
            running_value_loss += float(value_loss.item()) * x.size(0)
            preds = logits.argmax(dim=1)
            correct += (preds == y).sum().item()
            total += x.size(0)
    return running_loss/total, running_policy_loss/total, running_value_loss/total, correct/total

def load_trained(model_dir: str, device: torch.device, use_transformer: bool=True):
    with open(os.path.join(model_dir, 'move2idx.json'), 'r') as f:
        move2idx = json.load(f)
    idx2move = {int(idx): move for move, idx in move2idx.items()}
    num_moves = len(move2idx)
    model = ImprovedChessNet(num_moves, use_transformer=use_transformer, scalar_feat_dim=1).to(device)
    model.load_state_dict(torch.load(os.path.join(model_dir, 'best_model.pth'), map_location=device))
    model.eval()
    return model, move2idx, idx2move

def predict_move(fen: str, model, move2idx: Dict[str,int], idx2move: Dict[int,str], topk:int=5, legal_only:bool=True, device:torch.device=None):
    if device is None:
        device = next(model.parameters()).device
    x = mtools.fen_to_tensor(fen)
    x_t = torch.tensor(x, dtype=torch.float32).unsqueeze(0).to(device)
    r = torch.tensor([[2000.0/3000.0]], dtype=torch.float32).to(device)
    with torch.no_grad():
        logits, _ = model(x_t, scalar_feats=r)
        probs = torch.softmax(logits, dim=-1).cpu().numpy().reshape(-1)
    sorted_idx = np.argsort(-probs)
    board = chess.Board(fen)
    legal_set = set(m.uci() for m in board.legal_moves)
    results = []
    for idx in sorted_idx:
        move = idx2move.get(int(idx), None)
        if move is None:
            continue
        p = float(probs[idx])
        if legal_only:
            if move in legal_set:
                results.append((move,p))
        else:
            results.append((move,p))
        if len(results) >= topk:
            break
    if len(results) == 0 and legal_only:
        for idx in sorted_idx[:topk]:
            results.append((idx2move.get(int(idx),'?'), float(probs[idx])))
    return results