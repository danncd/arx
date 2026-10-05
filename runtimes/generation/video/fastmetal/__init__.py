import re
import subprocess
import sys
from pathlib import Path
from events import emit


def generate(request):
    emit("progress", progress=0, detail="Encoding video prompt")
    command = [
        sys.executable, "-u", str(Path(__file__).with_name("runner.py")),
        "--model-root", request["model_path"],
        "--mlx-checkpoint", request["model_path"],
        "--prompt", request["prompt"],
        "--output-path", request["output"],
        "--width", str(request["width"]),
        "--height", str(request["height"]),
        "--num-frames", str(request["frames"]),
        "--seed", str(request["seed"]),
        "--fps", "16",
        "--decode-backend", "wan-vae",
        "--torch-device", "mps",
        "--mlx-cache-limit-gib", "0.25",
        "--no-prompt-cache",
    ]
    with subprocess.Popen(command, stdout=subprocess.PIPE, text=True) as process:
        for line in process.stdout:
            print(line, end="", file=sys.stderr, flush=True)
            match = re.search(r"denoise step (\d+)/(\d+) complete", line)
            if match:
                completed, total = map(int, match.groups())
                emit(
                    "progress",
                    progress=0.2 + 0.65 * completed / total,
                    detail="Encoding video" if completed == total else "Generating video",
                )
        if process.wait() != 0:
            raise RuntimeError("FastMetal generation failed. Check the generation log")
    if not Path(request["output"]).is_file():
        raise RuntimeError("FastMetal did not produce a video")
    return {
        "mime": "video/mp4",
        "width": request["width"],
        "height": request["height"],
        "duration": request["frames"] / 16,
    }
