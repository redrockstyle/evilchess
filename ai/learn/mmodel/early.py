class EarlyStopping:
    def __init__(self, patience=5, min_delta=0, mode='min'):
        self.patience = patience
        self.min_delta = min_delta
        self.mode = mode
        self.counter = 0
        self.best_score = float('inf') if mode == 'min' else float('-inf')
        self.stop_training = False

        # debug
        self.val_loss_history = []

    def check(self, current_score):
        # debug
        self.val_loss_history.append(current_score)

        if self.mode == 'min':
            # better (if <min_delta)
            if current_score < self.best_score - self.min_delta:
                print(f"  [EarlyStopping] Score improved from {self.best_score:.4f} to {current_score:.4f}. Resetting counter.")
                self.best_score = current_score
                self.counter = 0
            else:
                # bad
                self.counter += 1
                print(f"  [EarlyStopping] Score did not improve. Counter: {self.counter}/{self.patience}")
        elif self.mode == 'max':
            # better (if >min_delta)
            if current_score > self.best_score + self.min_delta:
                print(f"  [EarlyStopping] Score improved from {self.best_score:.4f} to {current_score:.4f}. Resetting counter.")
                self.best_score = current_score
                self.counter = 0
            else:
                # bad
                self.counter += 1
                print(f"  [EarlyStopping] Score did not improve. Counter: {self.counter}/{self.patience}")
        
        if self.counter >= self.patience:
            self.stop_training = True
            print(f"  [EarlyStopping] Patience {self.patience} reached. Stopping training.")
            
        return self.stop_training