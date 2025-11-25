import argparse
import torch
from mmodel import model_normal as mmn
from mmodel import model_small as mms
from mtools import mtools

def main():
    parser = argparse.ArgumentParser(description='Chess Move Predictor')
    parser.add_argument('--outdir', type=str, required=True, help='Path to model')
    parser.add_argument('--no_transformer', action='store_true', help='Disable transformer block')
    parser.add_argument('-f','--fen', type=str, required=True, help='FEN string prediction')
    parser.add_argument('-r','--rating', type=int, required=True, help='Rating value prediction')
    parser.add_argument('--jit', action='store_true', help='Save JIT model')
    parser.add_argument('--onnx', action='store_true', help='Save ONNX model')
    parser.add_argument('--test_training', action='store_true', help='Use small model learning and prediction')
    
    args = parser.parse_args()

    device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
    print('Device: ', device)

    print(f'\nPrediction for {args.fen}:')
    trainer = mms.load(args.outdir+'_small', device)
    print(trainer.predict(args.fen, 3500))
    # if args.test_training:
    #     model, move2idx, idx2move = mms.load_trained(args.outdir+'_small', device)
    #     print(mms.predict_move(fen=args.fen, model=model, move2idx=move2idx, idx2move=idx2move, device=device))
    # else:
    #     model = mmn.load_trained(args.outdir+'_small', device, not args.no_transformer)
    #     print(mmn.predict_move(fen=args.fen, model=model, move2idx=move2idx, idx2move=idx2move, device=device))

    x_borad, x_rating = mtools.input_tensor(args.fen, args.rating)
    
    if args.jit:
        trainer.save_jit(args.outdir+'_small', x_borad, x_rating)
    
    if args.onnx:
        trainer.save_onnx(args.outdir+'_small', x_borad, x_rating)

    print('\nDone.')

if __name__ == "__main__":
    main()