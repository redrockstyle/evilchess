import argparse
import torch
import chess
import chess.pgn
from mmodel import model
from mtools import mtools

def main():
    parser = argparse.ArgumentParser(description='Chess Move Predictor')
    parser.add_argument('--outdir', type=str, required=True, help='Path to model')
    parser.add_argument('--use_transformer', action='store_true', help='Disable transformer block')
    parser.add_argument('--device', type=str, default='cuda', help='Device')
    parser.add_argument('--play', action='store_true', help='While SAN move')
    parser.add_argument('-f','--fen', type=str, required=True, help='FEN string prediction')
    parser.add_argument('-r','--rating', type=int, required=True, help='Rating value prediction')
    parser.add_argument('--jit', action='store_true', help='Save JIT model')
    parser.add_argument('--onnx', action='store_true', help='Save ONNX model')
    
    args = parser.parse_args()

    device = torch.device(args.device)
    print('Device: ', device)

    chess_model = model.ModelTrainer(device=device, model_dir=args.outdir)

    board = chess.Board(args.fen)
    game = chess.pgn.Game()
    game.setup(board)
    node = game

    if args.play:
        print("\n--- Interactive Chess Play ---")
        print("Enter moves in SAN format (e.g., e4, Nf3). Enter 'q' to quit.")

        while not board.is_game_over():
            print("\n" + "="*16)
            print(board.unicode(invert_color=True, empty_square="."))
            print(f"Current FEN: {board.fen()}")
            print(f"To Move: {'White' if board.turn == chess.WHITE else 'Black'}")
            print("\n" + "="*16)

            print("AI's top 5 predictions:")
            ai_predictions = chess_model.predict(board.fen(), args.rating, 5)
            for i, (move_uci, score) in enumerate(ai_predictions):
                try:
                    move_san = board.san(chess.Move.from_uci(move_uci))
                    print(f"  {i+1}. {move_san} (Score: {score:.4f})")
                except ValueError:
                    print(f"  {i+1}. {move_uci} (Score: {score:.4f}) - Invalid UCI for current board")

            user_input = input("Your move (SAN) or 'q' to quit: ").strip()

            if user_input.lower() == 'q':
                print("Exiting interactive play.")
                break

            try:
                move = board.parse_san(user_input)
                move_san_str = board.san(move)
                board.push(move)
                node = node.add_variation(move)
                print(f"You played: {move_san_str}")
            except ValueError as e:
                print(f"Invalid move: {e}. Please enter a legal SAN move.")
            except Exception:
                print(f"Invalid input: '{user_input}'. Please enter a legal SAN move.")

        print("\n--- Game Over ---")
        print(f"Result: {board.result()}")
        print("\n--- PGN Game Log ---")

        exporter = chess.pgn.StringExporter(headers=True, variations=True, comments=False)
        pgn_string = game.accept(exporter)
        print(pgn_string)

    else:
        print(f'\nPrediction for {args.fen}:')
        print(chess_model.predict(args.fen, args.rating, 10))

    if args.jit:
        chess_model.save_jit(args.outdir)
    if args.onnx:
        chess_model.save_onnx(args.outdir)

    print('\nDone.')

if __name__ == "__main__":
    main()