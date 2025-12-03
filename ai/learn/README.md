# AI Module

The main script, named [run.py](/ai/learn/run.py), sets up the required venv environment and uses it to run [mlearn_venv.py](/ai/learn/mlearn_venv.py) and [mpredict_venv.py](/ai/learn/mpredict_venv.py). There are used for training and testing the model. 

Use `run.py --help` for more details.

Example:
```
    python3 run.py predict --fen="6k1/5pp1/8/4p1Pp/b4q1P/2p2P2/2R5/3K1B2 b - - 5 47" --rating=3500 -y --outdir="chess_model_train10h"
``` 

```
    python .\run.py training --csv <PATH_TO_CSV_FILE> --epochs=1 --dataset_workers=8
``` 