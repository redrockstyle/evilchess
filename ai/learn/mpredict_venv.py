import argparse
import torch
from mmodel import model
from mtools import mtools

def main():
    parser = argparse.ArgumentParser(description='Chess Move Predictor')
    parser.add_argument('--outdir', type=str, required=True, help='Path to model')
    parser.add_argument('--use_transformer', action='store_true', help='Disable transformer block')
    parser.add_argument('--device', type=str, default='cuda', help='Device')
    parser.add_argument('-f','--fen', type=str, required=True, help='FEN string prediction')
    parser.add_argument('-r','--rating', type=int, required=True, help='Rating value prediction')
    parser.add_argument('--jit', action='store_true', help='Save JIT model')
    parser.add_argument('--onnx', action='store_true', help='Save ONNX model')
    
    args = parser.parse_args()

    # device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
    # device = torch.device('cpu')
    device = torch.device(args.device)
    print('Device: ', device)

    print(f'\nPrediction for {args.fen}:')
    chess_model = model.ModelTrainer(device=device, model_dir=args.outdir)
    print(chess_model.predict(args.fen, 3500, 10))

    x_borad, x_rating = mtools.input_tensor(args.fen, args.rating)
    if args.jit:
        chess_model.save_jit(args.outdir, x_borad, x_rating)
    # if args.onnx:
    #     trainer.save_onnx(args.outdir, x_borad, x_rating)

    print('\nDone.')

if __name__ == "__main__":
    main()