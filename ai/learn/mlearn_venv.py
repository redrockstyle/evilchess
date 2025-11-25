import argparse
import torch
from mtools import mlog
from mtools import mtools
from mmodel import dataset as ds
from mmodel import model_abc as mabc
from mmodel import model_normal as mmn
from mmodel import model_small as mms
from sklearn.model_selection import train_test_split
from torch.utils.data import DataLoader


def prepare_data(args):
    print(f'Prepare dataset {args.csv}...')
    df = mtools.load_csv(args.csv)
    print('Total rows:', len(df))

    # moves = df['move'].astype(str).tolist()
    # move2idx, idx2move = mtools.build_move_vocab(moves)
    # num_moves = len(move2idx)
    # print('Unique moves:', num_moves)

    train_df, test_df = train_test_split(df, test_size=args.test_size, random_state=42)
    print('Train:', len(train_df), 'Test:', len(test_df))

    train_ds = ds.ChessDataset(train_df, max_samples=args.max_samples)
    test_ds = ds.ChessDataset(test_df)
    return DataLoader(train_ds, batch_size=args.batch_size, shuffle=True, num_workers=args.num_workers, collate_fn=ds.collate_fn), \
        DataLoader(test_ds, batch_size=args.batch_size, shuffle=False, num_workers=args.num_workers, collate_fn=ds.collate_fn)


def prepare_model(args, device):
    return mms.SmallModelTrainer(args.lr, args.weight_decay, device)
    # if args.test_training:
    #     return mms.SmallModelTrainer(args.lr, args.weight_decay, device)
    # else:
    #     # scaler = torch.cuda.amp.GradScaler(enabled=(device.type=='cuda'))
    #     scaler = torch.amp.GradScaler('cuda', enabled=(device.type=='cuda'))
    #     return mmn.NormalModelTrainer(num_moves, not args.no_transformer,
    #                                     args.lr, args.weight_decay,
    #                                     scaler, args.alpha_value, device)


def training(args, trainer: mabc.ChessModelTrainer, train_loader, test_loader):
    print('Start learning...')
    
    trainer.init_scheduler(max(1, len(train_loader)), args.epochs)

    best_val_loss = 0.0
    for epoch in range(args.epochs):
        print(f'Epoch {epoch+1}/{args.epochs}')
        train_loss, train_pol_loss, train_val_loss, train_acc = trainer.train_one_epoch(train_loader)
        val_loss, val_pol_loss, val_val_loss, val_acc = trainer.eval(test_loader)
        print(f'  train_loss={train_loss:.4f} train_acc={train_acc:.4f} | val_loss={val_loss:.4f} val_acc={val_acc:.4f}')

        # save best
        if val_loss > best_val_loss:
            best_val_loss = val_loss
            trainer.save(args.outdir if not args.test_training else args.outdir + '_small')

            ## create one example
            (x_board_example, x_rating_example), _ = next(iter(train_loader))
            example_board = x_board_example[0].unsqueeze(0) # (1, 13, 8, 8)
            example_rating = x_rating_example[0].unsqueeze(0) # (1, 1)
            ## save model
            if args.jit:
                trainer.save_jit(args.outdir if not args.test_training else args.outdir + '_small', example_board, example_rating)
            # if args.onnx:
            #    trainer.save_onnx(args.outdir if not args.test_training else args.outdir + '_small', example_board, example_rating)
        
        trainer.save(args.outdir + f'_acc{val_acc}_loss{val_loss}' \
                     if not args.test_training else args.outdir + f'_acc{val_acc}_loss{val_loss}' + '_small')

    print('Training finished. Best val acc:', best_val_loss)


def pretty_print_args(args):
    args_dict = vars(args)
    max_key_len = max(len(k) for k in args_dict.keys())
    max_val_len = max(len(str(v)) for v in args_dict.values())
    def print_sep():
        print('+'+'-' * (max_key_len + max_val_len + 5)+'+')
    print_sep()
    print(f"| {'Parameter'.ljust(max_key_len)} | {'Value'.ljust(int(max_val_len))} |")
    print_sep()
    for key, val in args_dict.items():
        print(f"| {key.ljust(max_key_len)} | {str(val).ljust(max_val_len)} |")
    print_sep()


def ask_user(args, message: str) -> bool:
    if not args.yes:
        answer = input(f'{message}').strip().lower()
        if not answer in ("y", "yes"):
            print("Stoping...")
            return True
        print()
    return False 


def main():
    parser = argparse.ArgumentParser(description='Train Chess Move Predictor')
    parser.add_argument('--csv', type=str, required=True, help='Path to CSV dataset')
    parser.add_argument('--outdir', type=str, default='./model_out', help='Directory to save model')
    parser.add_argument('--batch_size', type=int, default=4096)
    parser.add_argument('--epochs', type=int, default=8)
    parser.add_argument('--lr', type=float, default=4e-4)
    parser.add_argument('--weight_decay', type=float, default=1e-5)
    parser.add_argument('--num_workers', type=int, default=8)
    parser.add_argument('--max_samples', type=int, default=0)
    parser.add_argument('--test_size', type=float, default=0.05)
    parser.add_argument('--no_transformer', action='store_true', help='Disable transformer block')
    parser.add_argument('--alpha_value', type=float, default=0.5, help='Weight for value loss')
    parser.add_argument('--test_training', action='store_true', help='Small model learning')
    parser.add_argument('-y','--yes', action='store_true', help='Automatically answers \"yes\"')
    parser.add_argument('--logfile', type=str, default='mlearn_venv.log', help='Custom logfile')
    args = parser.parse_args()

    mlog.start_logging(args.logfile, also_stderr=True)
    print("Preparation for training...")

    device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
    print('Device: ', device)
    pretty_print_args(args)
    print('Progress batch_size and lr:\n\t4096\t->\t4e-4\n\t8192\t->\t6e-4\n\t16384\t->\t8e-4\n\t32768\t->\t1e-3')

    train, test = prepare_data(args)
    trainer = prepare_model(args, device)
    
    print('\nTo monitor cuda performance, use: nvidia-smi -l 2')
    if ask_user(args, 'Everything necessary for training is prepared. Do you want to continue? [y/N]: '):
        return
    # training(args, model, move2idx, train, test, optimizer, scaler, loss_fn_policy, device)
    training(args, trainer, train, test)

    mlog.stop_logging()


if __name__ == "__main__":
    main()