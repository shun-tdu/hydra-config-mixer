class GRUEncoder:
    def __init__(self, input_dim: int, hidden_dim: int = 256, num_layers: int = 2, bidirectional: bool = True):
        self.input_dim = input_dim
        self.hidden_dim = hidden_dim
        self.num_layers = num_layers
        self.bidirectional = bidirectional


class LSTMEncoder:
    def __init__(self, input_dim: int, hidden_dim: int = 256, num_layers: int = 2, dropout: float = 0.1):
        self.input_dim = input_dim
        self.hidden_dim = hidden_dim
        self.num_layers = num_layers
        self.dropout = dropout


class TCNEncoder:
    def __init__(self, input_dim: int, num_channels: list = [64, 128, 256], kernel_size: int = 3, dropout: float = 0.2):
        self.input_dim = input_dim
        self.num_channels = num_channels
        self.kernel_size = kernel_size
        self.dropout = dropout
