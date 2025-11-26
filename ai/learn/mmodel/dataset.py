import pandas as pd
import numpy as np
import os
import torch
import chess
from tqdm import tqdm
from joblib import Parallel, delayed
from torch.utils.data import Dataset
from mtools import mtools


# ---------------------- Dataset ----------------------
class ChessDataset(Dataset):
    def __init__(self, df: pd.DataFrame, max_samples=0, num_workers=4, npy_board_tesors='temp_tensor_fens.npy', shared_drive=False): 
        # self.df = df.reset_index(drop=True)
        if max_samples != 0 and max_samples < len(df):
            df = df.sample(n=max_samples, random_state=42).reset_index(drop=True)

        N = len(df)
        D_board = 13 * 8 * 8
        boards_bytes = N * D_board # irnore * sizeof(int8)
        boards_gb = boards_bytes / (1024**3)
        
        print(f"--- Dataset Memory Projection ---")
        print(f"Total samples: {N:,}")
        print(f"Single tensor elements (int8): {D_board}")
        print(f"Boards size in RAM: ~{boards_gb:.2f} GB (before system optimizations)")

        if os.path.exists(npy_board_tesors):
            print(f'Use {npy_board_tesors} as a FEN->Tensor dataframe')
            if shared_drive:
                self.fens = np.load(npy_board_tesors, mmap_mode='r')
            else:
                self.fens = np.array(np.load(npy_board_tesors, mmap_mode=None), dtype=np.int8) # load all dataframe-tensor (bc use mmap_mode=None)
        else:
            print(f'--- Pre-processing FEN to Tensor (using {num_workers} processes) ---')
            fen_list = df['fen'].tolist()
            
            # chanks
            batch_size = 50000
            fen_chunks = [fen_list[i:i + batch_size] for i in range(0, len(fen_list), batch_size)]

            def process_batch(batch):
                return [mtools.fen_to_tensor(f) for f in batch]

            # parallel
            results = Parallel(n_jobs=num_workers, return_as="generator")(
                delayed(process_batch)(chunk) for chunk in tqdm(fen_chunks, desc="FEN->Tensor")
            )
            all_board_tensors = []
            for batch_res in results:
                all_board_tensors.extend(batch_res)

            ar_fens = np.concatenate(all_board_tensors, dtype=np.int8)
            # ar_fens = np.array(all_board_tensors, dtype=np.int8)
            np.save(npy_board_tesors, ar_fens)
            print(f'FEN->Tensor saved in {npy_board_tesors} (will be used in the future to skip pre-processing)')

            if shared_drive:
                self.fens = np.load(npy_board_tesors, mmap_mode='r')
            else:
                self.fens = ar_fens
            del all_board_tensors

        # fix OSError: [Errno 22] Invalid argument
        self.moves = df['move'].tolist()
        self.ratings = df['rating'].to_numpy(dtype=np.float32)
        self.results = df['result'].to_numpy(dtype=np.float32)

    def __len__(self):
        return len(self.moves)

    def __getitem__(self, idx):
        # 1. Board State (Tensor) - input
        # fix OSError: [Errno 22] Invalid argument
        x_board = self.fens[idx].astype(np.float32)
        move_str = self.moves[idx]
        rating = self.ratings[idx]
        result = self.results[idx]

        # 2. Extra Features (Rating) - input
        x_extra = np.array([rating / 3500.0], dtype=np.float32)

        # 3. Move (Target)
        try:
            move_obj = chess.Move.from_uci(move_str)
            y_from = move_obj.from_square   # (From-Square): 0-63
            y_to = move_obj.to_square       # (To-Square): 0-63
        except ValueError:
            # print(f"Warning: Invalid move {move_str} (skip move)")
            y_from = 0
            y_to = 0

        # 4. Result (Target)
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