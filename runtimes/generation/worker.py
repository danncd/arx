import json
import os
import sys
from events import emit


def main():
    os.environ["HF_HUB_OFFLINE"] = "1"
    os.environ["TRANSFORMERS_OFFLINE"] = "1"
    request = json.load(sys.stdin)
    operation = request["operation"]
    if operation == "image":
        if request.get("runtime") in ("flux2", "z-image"):
            from image.mlx import generate
        else:
            from image.diffusion import generate
    elif operation == "speech":
        from speech.synthesis import generate
    elif operation == "video":
        if request.get("runtime") == "fastmetal":
            from video.fastmetal import generate
        elif request.get("runtime") == "wan-cpp":
            from video.wan import generate
        elif request.get("runtime") == "animatediff":
            from video.animatediff import generate
        else:
            from video.ltx import generate
    elif operation == "combine":
        from media.combine import generate
    else:
        raise ValueError("Unsupported generation operation")
    result = generate(request)
    emit("complete", **result)


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        emit("error", detail=str(error))
        sys.exit(1)
