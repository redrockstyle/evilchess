import matplotlib.pyplot as plt
# import numpy as np
import os

def plot_training_history(history, save_dir='plots'):
    """
    Draw graphics for train/val loss & train/val accuracy.
    :param history: {'train_loss': [...], 'val_loss': [...], 'train_acc': [...], 'val_acc': [...]}
    """

    print(f'LEN train_loss  : {len(history['train_loss'])}')
    print(f'LEN val_loss    : {len(history['val_loss'])}')
    print(f'LEN train_acc   : {len(history['train_loss'])}')
    print(f'LEN val_acc     : {len(history['train_loss'])}')
    epochs = range(1, len(history['train_loss']) + 1)

    plt.figure(figsize=(12, 5))

    # Loss
    plt.subplot(1, 2, 1) # 1 ряд, 2 колонки, 1-й график
    plt.plot(epochs, history['train_loss'], 'b-', label='Train Loss')
    plt.plot(epochs, history['val_loss'], 'r-', label='Validation Loss')
    plt.title('Training and Validation Loss')
    plt.xlabel('Epochs')
    plt.ylabel('Loss')
    plt.legend()
    plt.grid(True)

    # Accuracy
    plt.subplot(1, 2, 2) # 1 ряд, 2 колонки, 2-й график
    plt.plot(epochs, history['train_acc'], 'b-', label='Train Accuracy')
    plt.plot(epochs, history['val_acc'], 'r-', label='Validation Accuracy')
    plt.title('Training and Validation Accuracy')
    plt.xlabel('Epochs')
    plt.ylabel('Accuracy')
    plt.legend()
    plt.grid(True)

    # auto layout
    plt.tight_layout()

    # save
    os.makedirs(save_dir, exist_ok=True)
    plt.savefig(os.path.join(save_dir, 'training_history.png'))
    plt.show()

if __name__ == '__main__':
    history_data = {
        "train_loss":[6.4303, 5.1197, 4.5259, 4.1138, 3.7635, 3.5071, 3.3246, 3.1834, 3.0717, 2.9817, 2.9093, 2.8506, 2.8010, 2.7590, 2.7213, 2.6884, 2.6594, 2.6349, 2.6127, 2.5926, 2.5746, 2.5578, 2.5423, 2.5276, 2.5134, 2.5002, 2.4868, 2.4738, 2.4614, 2.4486, 2.4359, 2.4235, 2.4111, 2.3985, 2.3857, 2.3732, 2.3608, 2.3484, 2.3361, 2.3242, 2.3128, 2.3021, 2.2918, 2.2829, 2.2747, 2.2677],
        "train_acc":[0.0501, 0.1163, 0.1630, 0.2047, 0.2443, 0.2740, 0.2964, 0.3156, 0.3321, 0.3463, 0.3583, 0.3688, 0.3780, 0.3856, 0.3928, 0.3991, 0.4046, 0.4094, 0.4138, 0.4177, 0.4211, 0.4243, 0.4274, 0.4302, 0.4329, 0.4355, 0.4378, 0.4401, 0.4424, 0.4446, 0.4467, 0.4487, 0.4507, 0.4528, 0.4548, 0.4567, 0.4585, 0.4604, 0.4622, 0.4639, 0.4655, 0.4670, 0.4682, 0.4697, 0.4707, 0.4716],
        "val_loss":[5.4558, 4.7014, 4.2135, 3.8798, 3.6065, 3.3434, 3.2708, 3.1691, 3.0505, 2.9540, 2.8721, 2.8130, 2.7752, 2.7493, 2.7029, 2.6777, 2.6623, 2.6429, 2.6245, 2.6157, 2.6024, 2.6002, 2.5773, 2.5621, 2.5753, 2.5561, 2.5471, 2.5392, 2.5319, 2.5250, 2.5218, 2.5134, 2.5101, 2.5154, 2.5066, 2.5014, 2.5076, 2.5063, 2.5068, 2.5031, 2.5010, 2.5075, 2.5045, 2.5049, 2.5089, 2.5105],
        "val_acc":[0.0984, 0.1504, 0.1945, 0.2305, 0.2670, 0.2967, 0.3015, 0.3163, 0.3374, 0.3532, 0.3699, 0.3765, 0.3817, 0.3909, 0.3964, 0.4046, 0.4104, 0.4090, 0.4163, 0.4170, 0.4208, 0.4264, 0.4273, 0.4301, 0.4330, 0.4339, 0.4348, 0.4380, 0.4382, 0.4407, 0.4419, 0.4434, 0.4434, 0.4449, 0.4455, 0.4473, 0.4475, 0.4487, 0.4500, 0.4495, 0.4501, 0.4512, 0.4507, 0.4516, 0.4513, 0.4518]
    }
    plot_training_history(history_data)