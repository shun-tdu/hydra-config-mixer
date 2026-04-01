class ActionClassifier:
    def __init__(self, num_classes: int, feature_dim: int = 256, dropout: float = 0.5, use_batchnorm: bool = True):
        self.num_classes = num_classes
        self.feature_dim = feature_dim
        self.dropout = dropout
        self.use_batchnorm = use_batchnorm
