from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="paw-sdk",
    version="1.0.0",
    author="PAW Chain",
    author_email="dev@paw.network",
    description="Official Python SDK for PAW blockchain",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://example.com/paw-chain/paw",
    packages=find_packages(),
    classifiers=[
        "Development Status :: 5 - Production/Stable",
        "Intended Audience :: Developers",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
    ],
    python_requires=">=3.8",
    install_requires=[
        "httpx>=0.24.0",
        "bech32>=1.2.0",
        "ecdsa>=0.18.0",
        "mnemonic>=0.20",
        "protobuf>=4.23.0",
        "cosmospy-protobuf>=1.0.0",
        "python-dateutil>=2.8.0",
    ],
    extras_require={
        "dev": [
            "pytest>=7.4.0",
            "pytest-asyncio>=0.21.0",
            "pytest-cov>=4.1.0",
            "black>=23.7.0",
            "mypy>=1.4.0",
            "pylint>=2.17.0",
        ],
    },
    keywords="paw blockchain cosmos sdk defi dex",
    project_urls={
        "Bug Reports": "https://example.com/paw-chain/paw/issues",
        "Source": "https://example.com/paw-chain/paw",
        "Documentation": "https://docs.paw.network",
    },
)
