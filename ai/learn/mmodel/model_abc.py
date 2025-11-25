from abc import ABC, abstractmethod
import torch
from typing import Dict, Tuple

class ChessModelTrainer(ABC):
    @abstractmethod
    def init_scheduler(self, steps_per_epoch: int, epochs: int):
        pass

    @abstractmethod
    def train_one_epoch(self, dataset) \
        -> Tuple[float, float, float, float]:
        pass

    @abstractmethod
    def eval(self, dataset) \
        -> Tuple[float, float, float, float]:
        pass

    @abstractmethod
    def save(self, out_dir):
        pass

    @abstractmethod
    def save_jit(self, out_dir, example_x_board: torch.Tensor, example_x_rating: torch.Tensor):
        pass

    @abstractmethod
    def save_onnx(self, out_dir, example_x_board: torch.Tensor, example_x_rating: torch.Tensor):
        pass

    @abstractmethod
    def predict(self, fen: str, idx2move: Dict[int,str], \
                topk:int=5, legal_only:bool=True) -> Dict[int]:
        pass