import pandas as pd
import numpy as np
import torch
import chess
from torch.utils.data import Dataset
from typing import Dict
from mtools import mtools

# ---------------------- Dataset ----------------------
class ChessDataset(Dataset):
    def __init__(self, df: pd.DataFrame, max_samples=0): 
        # self.df = df.reset_index(drop=True)
        if max_samples != 0 and max_samples < len(self.df):
            df = df.sample(n=min(max_samples, len(self.df)), random_state=42).reset_index(drop=True)

        # fix OSError: [Errno 22] Invalid argument
        self.fens = df['fen'].tolist()
        self.moves = df['move'].tolist()
        self.ratings = df['rating'].to_numpy(dtype=np.float32)
        self.results = df['result'].to_numpy(dtype=np.float32)

    def __len__(self):
        # return len(self.df)
        return len(self.fens)

    def __getitem__(self, idx):
        # fix OSError: [Errno 22] Invalid argument
        fen = self.fens[idx]
        move_str = self.moves[idx]
        rating = self.ratings[idx]
        result = self.results[idx]

        # row = self.df.iloc[idx]
        
        # 1. Board State (Tensor) - input
        # fen = row['fen']
        x_board = mtools.fen_to_tensor(fen) 

        # 2. Extra Features (Rating) - input
        # rating = row['rating']
        x_extra = np.array([rating / 3500.0], dtype=np.float32)

        # 3. Move (Target)
        # move_str = row['move']
        try:
            move_obj = chess.Move.from_uci(move_str)
            # (From-Square): 0-63
            y_from = move_obj.from_square 
            # (To-Square): 0-63
            y_to = move_obj.to_square
        except ValueError:
            print(f"Warning: Invalid move {move_str} in FEN {fen}. Skipping.")
            y_from = 0
            y_to = 0

        # 4. Result (Target)
        # result = row['result']
        y_result = np.array([result], dtype=np.float32)

        # (Move_From, Move_To, Result)
        return (torch.tensor(x_board), torch.tensor(x_extra)), \
               (torch.tensor(y_from, dtype=torch.long), 
                torch.tensor(y_to, dtype=torch.long), 
                torch.tensor(y_result))


def collate_fn(batch):
    # item: ((board, extra), (move_from, move_to, result))
    
    boards = []
    extras = []
    moves_from = []
    moves_to = []
    results = []

    for item in batch:
        (board, extra), (y_from, y_to, res) = item
        boards.append(board)
        extras.append(extra)
        moves_from.append(y_from)
        moves_to.append(y_to)
        results.append(res)

    x_board_batch = torch.stack(boards)      # (Batch, 13, 8, 8)
    x_extra_batch = torch.stack(extras)      # (Batch, 1)
    
    y_from_batch = torch.stack(moves_from)   # (Batch)
    y_to_batch = torch.stack(moves_to)       # (Batch)
    y_result_batch = torch.stack(results)    # (Batch, 1)

    # (Inputs), (Targets)
    return (x_board_batch, x_extra_batch), (y_from_batch, y_to_batch, y_result_batch)