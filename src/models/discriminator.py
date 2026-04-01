class MotionDiscriminator:
    def __init__(self, input_dim: int, hidden_dims: list = [256, 128], use_spectral_norm: bool = True, leaky_relu_slope: float = 0.2):
        self.input_dim = input_dim
        self.hidden_dims = hidden_dims
        self.use_spectral_norm = use_spectral_norm
        self.leaky_relu_slope = leaky_relu_slope
