from dataclasses import dataclass


@dataclass
class Account:
    address: str
    private_key: str
    password: str


user_1 = Account(
    address="0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
    private_key="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
    password="Strong12345"
)
user_2 = Account(
    address="0x70997970c51812dc3a010c7d01b50e0d17dc79c8",
    private_key="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
    password="Strong12345"
)
