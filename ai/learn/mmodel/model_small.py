"""
 - Самая простая модель SmallConvNet для тестовых обучений
"""

import torch
import os
import json
import numpy as np
import chess
import torch.nn as nn
import pandas as pd
from tqdm import tqdm
from mtools import mtools
from torch.utils.data import Dataset
from typing import Dict

# ---------------------- Dataset ----------------------
class ChessDataset(Dataset):
    def __init__(self, df: pd.DataFrame, move2idx: Dict[str,int], max_samples=0):
        self.df = df.reset_index(drop=True)
        if max_samples != 0 and max_samples < len(self.df):
            self.df = self.df.sample(n=min(max_samples,len(self.df)), random_state=42).reset_index(drop=True)
        self.move2idx = move2idx

    def __len__(self):
        return len(self.df)

    def __getitem__(self, idx):
        row = self.df.iloc[idx]
        fen = row['fen']
        move = row['move']
        x = mtools.fen_to_tensor(fen)  # (13,8,8)
        # target
        y = self.move2idx.get(move, -1)
        if y == -1:
            # если ход оказался незнаком — заменяем на 0 (или можно пропустить до подготовки данных)
            y = 0
        return torch.tensor(x), torch.tensor(y, dtype=torch.long)

def collate_fn(batch):
    xs = torch.stack([item[0] for item in batch])
    ys = torch.stack([item[1] for item in batch])
    return xs, ys


class SmallConvNet(nn.Module):
    def __init__(self, num_moves:int):
        super().__init__()
        self.net = nn.Sequential(
            nn.Conv2d(13, 64, kernel_size=3, padding=1),
            nn.BatchNorm2d(64),
            nn.ReLU(),
            nn.Conv2d(64, 128, kernel_size=3, padding=1),
            nn.BatchNorm2d(128),
            nn.ReLU(),
            nn.Conv2d(128, 128, kernel_size=3, padding=1),
            nn.BatchNorm2d(128),
            nn.ReLU(),
            nn.AdaptiveAvgPool2d((1,1)),
            nn.Flatten(),
            nn.Linear(128, 512),
            nn.ReLU(),
            nn.Dropout(0.3),
            nn.Linear(512, num_moves)
        )

    def forward(self, x):
        return self.net(x)
    

# ---------------------- Training ----------------------
def train_one_epoch(model, loader, opt, loss_fn, device):
    model.train()
    running_loss = 0.0
    correct = 0
    total = 0
    for x,y in tqdm(loader, desc='train', leave=False):
        x = x.to(device)
        y = y.to(device)
        opt.zero_grad()
        logits = model(x)
        loss = loss_fn(logits, y)
        loss.backward()
        opt.step()
        running_loss += float(loss.item()) * x.size(0)
        preds = logits.argmax(dim=1)
        correct += (preds == y).sum().item()
        total += x.size(0)
    return running_loss / total, correct/total


def eval_model(model, loader, loss_fn, device):
    model.eval()
    running_loss = 0.0
    correct = 0
    total = 0
    with torch.no_grad():
        for x,y in loader:
            x = x.to(device)
            y = y.to(device)
            logits = model(x)
            loss = loss_fn(logits, y)
            running_loss += float(loss.item()) * x.size(0)
            preds = logits.argmax(dim=1)
            correct += (preds == y).sum().item()
            total += x.size(0)
    return running_loss/total, correct/total


def load_trained(model_dir: str, device):
    with open(os.path.join(model_dir, 'move2idx.json'),'r') as f:
        move2idx = json.load(f)
    # idx2move = {int(v):k for k,v in enumerate(move2idx)} if False else {int(v):k for k,v in ((v,k) for k,v in move2idx.items())}
    # В move2idx json: {move:idx}, поэтому build idx2move properly
    idx2move = {int(idx):move for move,idx in move2idx.items()}
    num_moves = len(move2idx)
    model = SmallConvNet(num_moves).to(device)
    model.load_state_dict(torch.load(os.path.join(model_dir, 'best_model.pth'), map_location=device))
    model.eval()
    return model, move2idx, idx2move


def predict_move(fen: str, model, move2idx: Dict[str,int], idx2move: Dict[int,str], topk:int=5, legal_only:bool=True, device=None):
    if device is None:
        device = next(model.parameters()).device
    x = mtools.fen_to_tensor(fen)
    x_t = torch.tensor(x).unsqueeze(0).to(device)
    with torch.no_grad():
        logits = model(x_t)
        probs = torch.softmax(logits, dim=-1).cpu().numpy().reshape(-1)
    sorted_idx = np.argsort(-probs)
    # legal filter
    board = chess.Board(fen)
    legal_set = set(m.uci() for m in board.legal_moves)
    results = []
    for idx in sorted_idx:
        move = idx2move[idx]
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
            results.append((idx2move[idx], float(probs[idx])))
    return results