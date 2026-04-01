class MotionEncoder:
    def __init__(self, input_dim: int, output_dim: int = 128, num_layers: int = 3, activation: str = "relu"):
        self.input_dim = input_dim
        self.output_dim = output_dim
        self.num_layers = num_layers
        self.activation = activation
