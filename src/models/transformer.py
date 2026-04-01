class TransformerEncoder:
    def __init__(self, hidden_dim: int = 256, num_heads: int = 8, num_layers: int = 6, dropout: float = 0.1):
        self.hidden_dim = hidden_dim
        self.num_heads = num_heads
        self.num_layers = num_layers
        self.dropout = dropout
