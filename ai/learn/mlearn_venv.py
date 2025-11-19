import argparse
import torch
import pandas as pd
import torch.nn as nn
from mtools import mlog
from mtools import mtools
from mmodel import model_normal as mmn
from mmodel import model_small as mms
from sklearn.model_selection import train_test_split
from torch.utils.data import DataLoader


def prepare_data(args):
    print(f'Prepare dataset {args.csv}...')
    df = pd.read_csv(args.csv)
    df = df.dropna(subset=['move', 'fen', 'rating'])
    print('Total rows:', len(df))

    moves = df['move'].astype(str).tolist()
    move2idx, idx2move = mtools.build_move_vocab(moves)
    num_moves = len(move2idx)
    print('Unique moves:', num_moves)

    train_df, test_df = train_test_split(df, test_size=args.test_size, random_state=42)
    print('Train:', len(train_df), 'Test:', len(test_df))

    if args.test_learning:
        train_ds = mms.ChessDataset(train_df, move2idx, max_samples=args.max_samples)
        test_ds = mms.ChessDataset(test_df, move2idx)
        train_loader = DataLoader(train_ds, batch_size=args.batch_size, shuffle=True, num_workers=args.num_workers, collate_fn=mms.collate_fn)
        test_loader = DataLoader(test_ds, batch_size=args.batch_size, shuffle=False, num_workers=args.num_workers, collate_fn=mms.collate_fn)
    else:
        train_ds = mms.ChessDataset(train_df, move2idx, max_samples=args.max_samples)
        test_ds = mms.ChessDataset(test_df, move2idx, max_samples=args.max_samples and int(args.max_samples*0.1))
        train_loader = DataLoader(train_ds, batch_size=args.batch_size, shuffle=True, num_workers=args.num_workers, collate_fn=mmn.collate_fn)
        test_loader = DataLoader(test_ds, batch_size=args.batch_size, shuffle=False, num_workers=args.num_workers, collate_fn=mmn.collate_fn)
    return train_loader, test_loader, num_moves, move2idx, idx2move


def prepare_model(args, device, num_moves):
    # scaler = torch.cuda.amp.GradScaler(enabled=(device.type=='cuda'))
    scaler = torch.amp.GradScaler('cuda', enabled=(device.type=='cuda'))

    if args.test_learning:
        model = mms.SmallConvNet(num_moves).to(device)
        optimizer = torch.optim.Adam(
            model.parameters(), lr=args.lr, weight_decay=args.weight_decay)
        loss_fn_policy = nn.CrossEntropyLoss()
    else:
        model = mmn.ImprovedChessNet(
            num_moves,
            use_transformer=not args.no_transformer,
            scalar_feat_dim=1).to(device)
        optimizer = torch.optim.AdamW(
            model.parameters(), lr=args.lr, weight_decay=args.weight_decay)
        loss_fn_policy = nn.CrossEntropyLoss(label_smoothing=0.02)
    return model, optimizer, loss_fn_policy, scaler


def training(args, model, move2idx, train_loader, test_loader, optimizer, scaler, loss_fn_policy, device):
    print('Start learning...')
    # OneCycleLR requires total_steps or steps_per_epoch; we'll compute steps_per_epoch
    steps_per_epoch = max(1, len(train_loader))
    total_steps = steps_per_epoch * args.epochs
    scheduler = torch.optim.lr_scheduler.OneCycleLR(optimizer, max_lr=args.lr, total_steps=total_steps)

    best_val_acc = 0.0
    best_dir = None

    for epoch in range(args.epochs):
        print(f'Epoch {epoch+1}/{args.epochs}')
        if args.test_learning:
            train_loss, train_acc = mms.train_one_epoch(model, train_loader, optimizer, loss_fn_policy, device)
            val_loss, val_acc = mms.eval_model(model, test_loader, loss_fn_policy, device)
        else:
            train_loss, train_pol_loss, train_val_loss, train_acc = mmn.train_one_epoch(
                model, train_loader, optimizer, scaler, loss_fn_policy, args.alpha_value, device, scheduler=scheduler
            )
            val_loss, val_pol_loss, val_val_loss, val_acc = mmn.eval_model(model, test_loader, loss_fn_policy, args.alpha_value, device)
            print(f'  train_pol_loss={train_pol_loss:.4f} train_val_loss={train_val_loss:.4f} | val_pol_loss={val_pol_loss:.4f} val_val_loss={val_val_loss:.4f}')
        print(f'  train_loss={train_loss:.4f} train_acc={train_acc:.4f} | val_loss={val_loss:.4f} val_acc={val_acc:.4f}')

        # save best
        if val_acc > best_val_acc:
            best_val_acc = val_acc
            best_dir = args.outdir
            mtools.save_model(model, move2idx, args.outdir if not args.test_learning else args.outdir + '_small')

    print('Training finished. Best val acc:', best_val_acc)
    return best_dir


# def predict(test, model_path):
#     return


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
    parser.add_argument('--num_workers', type=int, default=4)
    parser.add_argument('--max_samples', type=int, default=0)
    parser.add_argument('--test_size', type=float, default=0.05)
    parser.add_argument('--no_transformer', action='store_true', help='Disable transformer block')
    parser.add_argument('--alpha_value', type=float, default=0.5, help='Weight for value loss')
    parser.add_argument('--test_learning', action='store_true', help='Small model learning')
    parser.add_argument('-y','--yes', action='store_true', help='Automatically answers \"yes\"')
    parser.add_argument('--logfile', type=str, default='mlearn_venv.log', help='Custom logfile')
    args = parser.parse_args()

    mlog.start_logging(args.logfile, also_stderr=True)
    print("Preparation for training...")

    device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
    print('Device: ', device)
    pretty_print_args(args)
    print('Progress batch_size and lr:\n\t4096\t->\t4e-4\n\t8192\t->\t6e-4\n\t16384\t->\t8e-4')

    train, test, num_moves, move2idx, idx2move = prepare_data(args)
    model, optimizer, loss_fn_policy, scaler = prepare_model(args, device, num_moves)
    
    print('\nTo monitor cuda performance, use: nvidia-smi -l 2')
    if ask_user(args, 'Everything necessary for training is prepared. Do you want to continue? [y/N]: '):
        return
    training(args, model, move2idx, train, test, optimizer, scaler, loss_fn_policy, device)

    # if ask_user(args, 'The model is trained. Do you want to evaluate it on test data? [y/N]: '):
    #     return
    # predict(test, model)

    mlog.stop_logging()


if __name__ == "__main__":
    main()