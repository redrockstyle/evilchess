"""
 - Model
"""

import os
import torch
import chess
import torch.nn as nn
from tqdm import tqdm
from mtools import mtools
from typing import List, Tuple


class TinyTransformer(nn.Module):
    def __init__(self, input_dim=192, model_dim=192, nhead=8, num_layers=2, dropout=0.1):
        super().__init__()
        
        # size
        self.embedding = nn.Linear(input_dim, model_dim)

        # 8x8 = 64
        self.pos_embed = nn.Parameter(torch.randn(1, 64, model_dim) * 0.02)
        encoder_layer = nn.TransformerEncoderLayer(
            d_model=model_dim,
            nhead=nhead,
            dim_feedforward=model_dim * 4, 
            dropout=dropout,
            activation='gelu', # GELU часто лучше ReLU в трансформерах
            batch_first=True, 
            # norm_first=True    # Pre-LN обычно стабильнее при обучении
        )
        self.transformer_encoder = nn.TransformerEncoder(encoder_layer, num_layers=num_layers)
        self.norm_final = nn.LayerNorm(model_dim)

    def forward(self, x):
        # (Batch, 64, input_dim)
        B, S, D = x.shape

        # Проекция (если input_dim != model_dim, здесь это Identity)
        x = self.embedding(x) 

        # Добавляем информацию о позиции (Broadcasting (1, 64, D) на (B, 64, D))
        x = x + self.pos_embed 

        # Прогоняем через слои трансформера
        x = self.transformer_encoder(x) # (B, 64, model_dim)
        x = self.norm_final(x)

        # Глобальный пулинг: Агрегируем информацию со всех 64 клеток в один вектор.
        # Усреднение по измерению последовательности (dim=1).
        x_pooled = x.mean(dim=1) # Результат: (B, model_dim)
        
        return x_pooled


class ResidualBlock(nn.Module):
    """
    Basic Residual Connections
    Input -> Conv1 -> BN1 -> ReLU -> Conv2 -> BN2 -> (+ Input) -> ReLU -> Output
    """
    def __init__(self, in_channels, out_channels, kernel_size=3, padding=1):
        super().__init__()
        
        # bias=False bc next BatchNorm
        self.conv1 = nn.Conv2d(in_channels, out_channels, kernel_size=kernel_size, 
                               padding=padding, bias=False)
        self.bn1 = nn.BatchNorm2d(out_channels)
        self.relu = nn.ReLU(inplace=True)
        
        self.conv2 = nn.Conv2d(out_channels, out_channels, kernel_size=kernel_size, 
                               padding=padding, bias=False)
        self.bn2 = nn.BatchNorm2d(out_channels)
        
        # shortcut/skip connection
        self.shortcut = nn.Identity()
        if in_channels != out_channels:
            self.shortcut = nn.Sequential(
                nn.Conv2d(in_channels, out_channels, kernel_size=1, bias=False),
                nn.BatchNorm2d(out_channels)
            )

    def forward(self, x):
        residual = x
        
        out = self.conv1(x)
        out = self.bn1(out)
        out = self.relu(out)
        
        out = self.conv2(out)
        out = self.bn2(out)
        
        # magic !!!
        out += self.shortcut(residual) 
        
        out = self.relu(out)
        return out


class ChessModel(nn.Module):
    def __init__(self, use_transformer: bool = False):
        super().__init__()
        self.use_transformer = use_transformer
        
        # 1. Feature Extractor (CNN)
        # можно увеличить для произоводительности kernel_size в первом слое до 5 (padding=2) или 7 (padding=3)
        cnn_output_channels = 192 # если менять, то менять и model_dim в трансформере
        self.conv_block = nn.Sequential(
            # Initial Stem
            # bias=False, т.к. дальше BN
            nn.Conv2d(13, 96, kernel_size=7, padding=3, bias=False), 
            nn.BatchNorm2d(96),
            nn.ReLU(inplace=True),

            ResidualBlock(in_channels=96, out_channels=128),
            ResidualBlock(in_channels=128, out_channels=128),
            # ResidualBlock(in_channels=128, out_channels=192),
            # ResidualBlock(in_channels=192, out_channels=192),
            ResidualBlock(in_channels=128, out_channels=192),
            ResidualBlock(in_channels=192, out_channels=cnn_output_channels)
        )
        # self.conv_block = nn.Sequential(
        #     nn.Conv2d(13, 96, kernel_size=7, padding=3),
        #     # nn.Conv2d(13, 64, kernel_size=3, padding=1),
        #     nn.BatchNorm2d(96),
        #     # nn.BatchNorm2d(64),
        #     nn.ReLU(inplace=True), # inplace=True экономит память
            
        #     nn.Conv2d(96, 128, kernel_size=3, padding=1),
        #     # nn.Conv2d(64, 128, kernel_size=3, padding=1),
        #     nn.BatchNorm2d(128),
        #     nn.ReLU(inplace=True),
            
        #     nn.Conv2d(128, 192, kernel_size=3, padding=1),
        #     # nn.Conv2d(128, 128, kernel_size=3, padding=1),
        #     nn.BatchNorm2d(192),
        #     # nn.BatchNorm2d(128),
        #     nn.ReLU(inplace=True),
            
        #     # можно попробовать эти три слоя исключить
        #     # если менять в этом слое веса, то нужно менять и в fc_common входящие веса
        #     nn.Conv2d(192, 256, kernel_size=3, padding=1),
        #     nn.BatchNorm2d(256),
        #     nn.ReLU(inplace=True),

        #     nn.Conv2d(256, 384, kernel_size=3, padding=1),
        #     nn.BatchNorm2d(384),
        #     nn.ReLU(inplace=True),

        #     nn.Conv2d(384, 512, kernel_size=3, padding=1),
        #     nn.BatchNorm2d(512),
        #     nn.ReLU(inplace=True),

        #     nn.Conv2d(512, 512, kernel_size=3, padding=1),
        #     nn.BatchNorm2d(512),
        #     nn.ReLU(inplace=True),

        #     nn.Conv2d(512, cnn_output_channels, kernel_size=3, padding=1),
        #     nn.BatchNorm2d(cnn_output_channels),
        #     nn.ReLU(inplace=True)
        # )
        
        if self.use_transformer:
            self.transformer = TinyTransformer(
                input_dim=cnn_output_channels,
                model_dim=cnn_output_channels, # было 192
                nhead=8, 
                num_layers=2,
                dropout=0.1
            )
            self.fc_input_dim = cnn_output_channels # было 192
        else:
            # Global Average Pooling ?? Flatten ??
            self.avg_pool = nn.AdaptiveAvgPool2d((1,1))
            self.flatten = nn.Flatten()
            self.fc_input_dim = cnn_output_channels

        # 2. Common Dense Layer
        # на вход fc_input_dim каналов от CNN + 1 число рейтинга
        self.fc_common = nn.Sequential(
            nn.Linear(self.fc_input_dim + 1, 512),
            # nn.Linear(128 + 1, 512),
            nn.ReLU(inplace=True),
            nn.Dropout(0.3)
        )

        # 3. Policy Heads
        # Head From Square: 64 клетки
        self.from_head = nn.Linear(512, 64)
        # Haed To Square: 64 клетки
        self.to_head = nn.Linear(512, 64)
        # Value Head: 1 число (победа/поражение)
        # вывод 1 число: Tanh дает от -1 (Black wins) до 1 (White wins).
        self.value_head = nn.Sequential(
            nn.Linear(512, 64),
            nn.ReLU(inplace=True),
            nn.Linear(64, 1) # активация будет в лоссе (BCEWithLogits)
        )

    def forward(self, x_board: torch.Tensor, x_rating: torch.Tensor) -> Tuple[torch.Tensor, torch.Tensor, torch.Tensor]:
        # x_board: (B, 13, 8, 8)
        # x_rating: (B, 1)

        # CNN
        feat = self.conv_block(x_board)

        B, C, H, W = feat.shape
        if self.use_transformer:
            # S=H*W=64
            # (B, C, H, W) -> (B, S, C)
            
            # сплющиваем пространственные измерения H и W: (B, C, 64)
            feat = feat.flatten(2) 
            
            # переставляем размерности, чтобы последовательность была в середине: (B, 64, C)
            feat = feat.permute(0, 2, 1) 
            
            # подаем в трансформер. Он вернет усредненный вектор.
            feat = self.transformer(feat) # out: (B, 192)
            
        else:
            feat = self.avg_pool(feat) # (B, 192, 1, 1)
            feat = self.flatten(feat)  # (B, 192)

        # conact rating and board
        combined = torch.cat([feat, x_rating], dim=1) # (Batch, 129)

        # полносвязный слой
        common = self.fc_common(combined)

        # heads
        out_from = self.from_head(common) # (Batch, 64)
        out_to = self.to_head(common)     # (Batch, 64)
        out_val = self.value_head(common) # (Batch, 1)

        return out_from, out_to, out_val
    

class ModelTrainer:
    def __init__(self, lr: float = 0.0, weight_decay: float = 0.0, use_transformer: bool = False, device: torch.device = None, model_dir: str = ''):
        self.device = device
        if model_dir == '':
            self.model = ChessModel(use_transformer=use_transformer).to(device)
            self.optimizer = torch.optim.Adam(self.model.parameters(), lr=lr, weight_decay=weight_decay)

            self.loss_ce = nn.CrossEntropyLoss()        # Для From и To
            # self.loss_mse = nn.BCEWithLogitsLoss()    # Для Value (можно попробовать вместо MSELoss)
            self.loss_mse = nn.MSELoss()                # Для Value

            # Mixed Precision
            self.scaler = torch.amp.GradScaler('cuda', enabled=(self.device.type == 'cuda'))
            self.scheduler = None
        else:
            model_path = os.path.join(model_dir, 'best_model_coords.pth')
            if not os.path.exists(model_path):
                raise FileNotFoundError(f"Model not found")
            self.model = ChessModel(use_transformer=use_transformer).to(device)
            self.model.load_state_dict(torch.load(model_path, map_location=device))
            self.model.eval()

    def init_scheduler(self, steps_per_epoch: int, epochs: int):
        self.scheduler = torch.optim.lr_scheduler.OneCycleLR(
            self.optimizer, max_lr=1e-3, steps_per_epoch=steps_per_epoch, epochs=epochs
        )

    def train_one_epoch(self, dataset) -> Tuple[float, float, float, float]:
        self.model.train()
        
        total_loss = 0.0
        total_acc = 0.0
        samples = 0
        
        # from collate_fn(batch)
        for (x_board, x_rating), (y_from, y_to, y_result) in tqdm(dataset, desc='train', leave=False):
            x_board, x_rating = x_board.to(self.device), x_rating.to(self.device)
            y_from, y_to, y_result = y_from.to(self.device), y_to.to(self.device), y_result.to(self.device)
            
            self.optimizer.zero_grad()
            
            with torch.amp.autocast('cuda', enabled=(self.device.type == 'cuda')):
                pred_from, pred_to, pred_val = self.model(x_board, x_rating)
                
                # calc loss
                l_from = self.loss_ce(pred_from, y_from)
                l_to = self.loss_ce(pred_to, y_to)
                l_val = self.loss_mse(pred_val, y_result)
                
                # summ loss: From + To + (weight * Value)
                loss = l_from + l_to + 0.5 * l_val

            self.scaler.scale(loss).backward()
            self.scaler.step(self.optimizer)
            self.scaler.update()
            
            if self.scheduler:
                self.scheduler.step()

            bs = x_board.size(0)
            total_loss += loss.item() * bs
            
            # accuracy
            acc_from = (pred_from.argmax(dim=1) == y_from)
            acc_to = (pred_to.argmax(dim=1) == y_to)
            full_acc = (acc_from & acc_to).float().sum().item()
            
            total_acc += full_acc
            samples += bs
            
        return total_loss / samples, 0.0, 0.0, total_acc / samples

    def eval(self, dataset) -> Tuple[float, float, float, float]:
        self.model.eval()
        total_loss = 0.0
        total_acc = 0.0
        samples = 0
        
        with torch.no_grad():
            for (x_board, x_rating), (y_from, y_to, y_result) in dataset:
                x_board, x_rating = x_board.to(self.device), x_rating.to(self.device)
                y_from, y_to, y_result = y_from.to(self.device), y_to.to(self.device), y_result.to(self.device)
                
                pred_from, pred_to, pred_val = self.model(x_board, x_rating)
                
                l_from = self.loss_ce(pred_from, y_from)
                l_to = self.loss_ce(pred_to, y_to)
                l_val = self.loss_mse(pred_val, y_result)
                loss = l_from + l_to + 0.5 * l_val
                
                bs = x_board.size(0)
                total_loss += loss.item() * bs
                
                acc_from = (pred_from.argmax(dim=1) == y_from)
                acc_to = (pred_to.argmax(dim=1) == y_to)
                total_acc += (acc_from & acc_to).float().sum().item()
                samples += bs
                
        return total_loss / samples, 0.0, 0.0, total_acc / samples

    def save(self, out_dir):
        os.makedirs(out_dir, exist_ok=True)
        torch.save(self.model.state_dict(), os.path.join(out_dir, 'best_model_coords.pth'))

    def save_jit(self, out_dir, example_x_board: torch.Tensor, example_x_rating: torch.Tensor):
        self.model.eval()
        try:
            os.makedirs(out_dir, exist_ok=True)

            input_board = example_x_board.to(self.device)
            input_rating = example_x_rating.to(self.device)
            traced_model = torch.jit.trace(self.model, (input_board, input_rating))
            
            os.makedirs(out_dir, exist_ok=True)
            traced_model.save(os.path.join(out_dir, 'best_model_jit.pt'))
            # print("Model saved successfully using torch.jit.trace.")

            # scd = torch.jit.script(self.model)
            # scd.save(os.path.join(out_dir, 'best_model.pt'))
        except Exception as e:
            print(f"JIT Tracing failed: {e}")

    def save_onnx(self, out_dir, example_x_board: torch.Tensor, example_x_rating: torch.Tensor):
        # self.model.eval()
        # input_tuple = (
        #     example_x_board.to(self.device).unsqueeze(0),
        #     example_x_rating.to(self.device).unsqueeze(0)
        # )

        # input_names = ["board_input", "rating_input"]
        # output_names = ["policy_from", "policy_to", "value_output"]
    
        # torch.onnx.export(
        #     self.model,
        #     input_tuple, # input
        #     os.path.join(out_dir, "best_model.onnx"),
        #     export_params=True,
        #     opset_version=14, # version
        #     do_constant_folding=True,
        #     input_names=input_names,
        #     output_names=output_names,
        #     dynamic_axes=None
        # )
        # print(f"Model successfully exported to ONNX: {os.path.join(out_dir, 'best_model.onnx')}")
        pass

    def predict(self, fen: str, rating: float = 2500.0, topk: int = 5) -> List[Tuple[str, float]]:
        """
        Predict legal move by FEN
        """
        # 1. Prepare
        board = chess.Board(fen)
        x_board = mtools.fen_to_tensor(fen)
        x_board = torch.tensor(x_board, dtype=torch.float32).unsqueeze(0).to(self.device) # (1, 13, 8, 8)
        
        x_rating = torch.tensor([[rating / 3500.0]], dtype=torch.float32).to(self.device)
        
        self.model.eval()
        with torch.no_grad():
            # Получаем логиты (сырые вероятности)
            logits_from, logits_to, val_pred = self.model(x_board, x_rating)
            
            # Softmax для получения вероятностей
            probs_from = torch.softmax(logits_from, dim=1).squeeze(0).cpu().numpy() # (64,)
            probs_to = torch.softmax(logits_to, dim=1).squeeze(0).cpu().numpy()     # (64,)
            
        # 2. Legal Moves
        legal_moves_scores = []
        for move in board.legal_moves:
            from_idx = move.from_square
            to_idx = move.to_square
            
            # Считаем "очки" хода как произведение вероятностей
            score = probs_from[from_idx] * probs_to[to_idx]
            legal_moves_scores.append((move.uci(), score))
            
        # 3. Сортировка и выдача Top-K
        legal_moves_scores.sort(key=lambda x: x[1], reverse=True)
        
        return legal_moves_scores[:topk]
    

# def load(model_dir: str, device: str) -> ModelTrainer:
#         model_path = os.path.join(model_dir, 'best_model_coords.pth')
#         if not os.path.exists(model_path):
#             raise FileNotFoundError(f"Model not found")
#         model = ChessModel().to(device)
#         model.load_state_dict(torch.load(model_path, map_location=device))
#         model.eval()
#         return ModelTrainer(model=model, device=device)