import argparse
import torch
import chess
import numpy as np
from mmodel import model_normal as mmn
from mmodel import model_small as mms
from typing import Dict
from mtools import mtools


def main():
    parser = argparse.ArgumentParser(description='Chess Move Predictor')
    parser.add_argument('--outdir', type=str, required=True, help='Path to model')
    parser.add_argument('--no_transformer', action='store_true', help='Disable transformer block')
    parser.add_argument('-f','--fen', type=str, required=True, help='FEN string prediction')
    parser.add_argument('--test_learning', action='store_true', help='Use small model learning and prediction')
    
    args = parser.parse_args()

    device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
    print('Device: ', device)

    print(f'\nPrediction for {args.fen}:')
    if args.test_learning:
        model, move2idx, idx2move = mms.load_trained(args.outdir+'_small', device)
        print(mms.predict_move(fen=args.fen, model=model, move2idx=move2idx, idx2move=idx2move, device=device))
    else:
        model = mmn.load_trained(args.outdir+'_small', device, not args.no_transformer)
        print(mmn.predict_move(fen=args.fen, model=model, move2idx=move2idx, idx2move=idx2move, device=device))

    print('\nDone.')

if __name__ == "__main__":
    main()