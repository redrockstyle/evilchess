"""
Utils
"""

import json
import torch
import os
import chess
import pandas as pd
import numpy as np
from typing import List, Tuple, Dict

piece_to_plane = {
    'P':0,'N':1,'B':2,'R':3,'Q':4,'K':5,
    'p':6,'n':7,'b':8,'r':9,'q':10,'k':11
}

def fen_to_tensor(fen: str) -> np.ndarray:
    """
    Convert FEN to numpy array (13, 8, 8).
    Planes 0-5: White pieces
    Planes 6-11: Black pieces
    Plane 12: Side to move (All 1.0 if White, All 0.0 if Black)
    """
    board = chess.Board(fen)
    planes = np.zeros((13,8,8), dtype=np.float32)
    # pieces
    for square in chess.SQUARES:
        piece = board.piece_at(square)
        if piece is not None:
            plane_idx = piece_to_plane[piece.symbol()]
            # chess.square_rank возвращает 0 для 1-й горизонтали. 
            # чтобы визуально было как на доске (сверху вниз), делаем 7 - rank.
            row = 7 - chess.square_rank(square)
            col = chess.square_file(square)
            planes[plane_idx, row, col] = 1.0

    # side to move plane
    if board.turn == chess.WHITE:
        planes[12, :, :] = 1.0
    return planes


def load_csv(path: str) -> pd.DataFrame:
    df = pd.read_csv(path)
    # check format dataset
    assert {'id','rating','side','result','halfmove','move','fen'}.issubset(df.columns), 'CSV missing columns'
    # skip
    df = df.dropna(subset=['rating', 'result', 'move','fen'])
    return df


def build_move_vocab(moves: List[str]) -> Tuple[Dict[str,int], Dict[int,str]]:
    uniq = sorted(list(set(moves)))
    move2idx = {m:i for i,m in enumerate(uniq)}
    idx2move = {i:m for m,i in move2idx.items()}
    return move2idx, idx2move

def save_model_pth(model, move2idx, out_dir):
    os.makedirs(out_dir, exist_ok=True)
    torch.save(model.state_dict(), os.path.join(out_dir, 'best_model.pth'))
    with open(os.path.join(out_dir, 'move2idx.json'), 'w') as f:
        json.dump(move2idx, f)


def save_model_pt(model, move2idx, out_dir):
    os.makedirs(out_dir, exist_ok=True)
    scripted_model = torch.jit.script(model)
    scripted_model.save(os.path.join(out_dir, 'best_model.pt'))
    with open(os.path.join(out_dir, 'move2idx.json'), 'w') as f:
        json.dump(move2idx, f)

def input_tensor(fen: str, rating: int) -> Tuple[torch.Tensor, torch.Tensor]:
    # board: (13, 8, 8) -> (1, 13, 8, 8)
    x_board = torch.tensor(fen_to_tensor(fen)).unsqueeze(0)
    # rating: (1, 1)
    x_rating = torch.tensor([[rating / 3500.0]], dtype=torch.float32)
    return x_board, x_rating