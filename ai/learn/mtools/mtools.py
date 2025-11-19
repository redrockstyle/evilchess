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
    """Convert FEN to numpy array (13,8,8)"""
    board = chess.Board(fen)
    planes = np.zeros((13,8,8), dtype=np.float32)
    # pieces
    for square in chess.SQUARES:
        piece = board.piece_at(square)
        if piece is not None:
            plane_idx = piece_to_plane[piece.symbol()]
            row = 7 - chess.square_rank(square)  # делаем 0 - верхняя строка (8-я), для визуальной удобности
            col = chess.square_file(square)
            planes[plane_idx, row, col] = 1.0
    # side to move plane
    side_plane = np.ones((8,8), dtype=np.float32) if board.turn == chess.WHITE else np.zeros((8,8), dtype=np.float32)
    planes[12] = side_plane
    return planes


def load_csv(path: str) -> pd.DataFrame:
    df = pd.read_csv(path)
    assert {'id','rating','side','result','halfmove','move','fen'}.issubset(df.columns), 'CSV missing columns'
    # skip
    df = df.dropna(subset=['move','fen'])
    return df


def build_move_vocab(moves: List[str]) -> Tuple[Dict[str,int], Dict[int,str]]:
    uniq = sorted(list(set(moves)))
    move2idx = {m:i for i,m in enumerate(uniq)}
    idx2move = {i:m for m,i in move2idx.items()}
    return move2idx, idx2move

def save_model(model, move2idx, out_dir):
    os.makedirs(out_dir, exist_ok=True)
    torch.save(model.state_dict(), os.path.join(out_dir, 'best_model.pth'))
    with open(os.path.join(out_dir, 'move2idx.json'), 'w') as f:
        json.dump(move2idx, f)