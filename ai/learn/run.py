import argparse
import subprocess
from mtools import mvenv
from mtools import mlog

def sync_exec(py_venv: str, script: str, *args):
    process = subprocess.Popen([py_venv, script, *args])
    process.wait()


def main():
    parser = argparse.ArgumentParser(description='Train Chess Move Predictor')
    parser.add_argument('command', choices=['training', 'predict'], help='Mod execution')
    parser.add_argument('--fen', type=str, default='', help='FEN for predict')
    parser.add_argument('--csv', type=str, default='', help='Path to CSV dataset')
    parser.add_argument('--venv_learn', type=str, default="mlearn_venv.py", help='Script for training')
    parser.add_argument('--venv_predict', type=str, default="mpredict_venv.py", help='Script for prediction')
    parser.add_argument('--outdir', type=str, default='./model_out', help='Directory to save model')
    parser.add_argument('--batch_size', type=int, default=4096)
    parser.add_argument('--epochs', type=int, default=8)
    parser.add_argument('--lr', type=float, default=4e-4)
    parser.add_argument('--weight_decay', type=float, default=1e-5)
    parser.add_argument('--num_workers', type=int, default=8, help='Number of workers for prepare dataset')
    parser.add_argument('--max_samples', type=int, default=0)
    parser.add_argument('--test_size', type=float, default=0.05, help='Split dataset')
    parser.add_argument('--no_transformer', action='store_true', help='Disable transformer block')
    parser.add_argument('--alpha_value', type=float, default=0.5, help='Weight for value loss')
    parser.add_argument('--test_learning', action='store_true', help='Use small model learning and prediction')
    parser.add_argument('-y','--yes', action='store_true', help='Automatically answers \"yes\"')
    parser.add_argument('--logfile', type=str, default='mlearn.log', help='Custom logfile')
    args = parser.parse_args()

    mlog.start_logging(args.logfile, also_stderr=True)
    print("Starting...")
    status, path, active, pypath = mvenv.open_venv(venv_path="venv", prompt_user=args.yes)
    if not status:
        mlog.stop_logging()
        return
    print(f"VENV:\
          \n\tPath   → {path}\
          \n\tActive → {active}\
          \n\tPython → {pypath}\n")
    
    if not args.yes:
        answer = input("Python venv has been successfully installed. Do you want to continue? [y/N]: ").strip().lower()
        if not answer in ("y", "yes"):
            print("Bye!")
            return
        print()

    arg_list = []
    venv_script = ''
    if args.command == 'training':
        if args.csv == '':
            print('CSV path is not defined (use --csv)')
            return
        venv_script = args.venv_learn
        for key, val in vars(args).items():
            if key in {'logfile', 'venv_learn', 'venv_predict', 'command', 'fen'}:
                continue
            if isinstance(val, bool):
                if val:
                    arg_list.append(f"--{key}")
            else:
                arg_list.append(f"--{key}")
                arg_list.append(str(val))
    elif args.command == 'predict':
        if args.fen == '':
            print('FEN string is not defined (use --fen)')
            return
        venv_script = args.venv_predict
        for key, val in vars(args).items():
            if key in {'fen', 'outdir', 'test_learning', 'no_transformer'}:
                if isinstance(val, bool):
                    if val:
                        arg_list.append(f"--{key}")
                else:
                    arg_list.append(f"--{key}")
                    arg_list.append(str(val))

    print(f'Executing {venv_script}...')
    sync_exec(pypath, venv_script, *arg_list)

    mlog.stop_logging()


if __name__ == "__main__":
    main()



