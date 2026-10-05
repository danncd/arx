import runpy
import sys
import types
from pathlib import Path


def main():
    source = Path(sys.prefix).parent / "source"
    for name in (
        "fastvideo",
        "fastvideo.mlx_runtime",
        "fastvideo.models",
        "fastvideo.models.schedulers",
    ):
        module = types.ModuleType(name)
        module.__path__ = [str(source.joinpath(*name.split(".")))]
        sys.modules[name] = module
    runpy.run_path(
        str(source / "examples/inference/basic/mlx_wan_prompt_to_video.py"),
        run_name="__main__",
    )


if __name__ == "__main__":
    main()
