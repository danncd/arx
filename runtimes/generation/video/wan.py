import re
import subprocess
import sys
import tempfile
from pathlib import Path
from events import emit


def command(request, output):
    model = Path(request["model_path"])
    diffusion = model / "diffusion.gguf"
    if not diffusion.is_file():
        diffusion = model / "wan2.1_t2v_1.3B_fp16.safetensors"
    runtime = Path(sys.prefix).parent / "native" / "sd-cli"
    return [
        str(runtime), "-M", "vid_gen",
        "--diffusion-model", str(diffusion),
        "--vae", str(model / "wan_2.1_vae.safetensors"),
        "--t5xxl", str(model / "umt5-xxl-encoder-Q4_K_M.gguf"),
        "-p", request["prompt"],
        "-n", "blurry, distorted, text, watermark",
        "--cfg-scale", "6", "--sampling-method", "euler",
        "--steps", str(request["steps"]),
        "-W", str(request["width"]), "-H", str(request["height"]),
        "--video-frames", str(request["frames"]), "--flow-shift", "3",
        "--diffusion-fa", "--backend", "metal",
        "--params-backend", "diffusion=metal,te=disk,vae=disk",
        "--max-vram", "8", "--temporal-tiling", "--vae-tiling",
        "-s", str(request["seed"]), "-o", str(output),
    ]


def run(request, output):
    emit("progress", progress=0, detail="Encoding video prompt")
    with subprocess.Popen(
        command(request, output), stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT, text=True, errors="replace",
    ) as process:
        for line in process.stdout:
            print(line, end="", file=sys.stderr, flush=True)
            match = re.search(r"\|\s*(\d+)/(\d+) - [\d.]+(?:s/it|it/s)", line)
            if match:
                completed, total = map(int, match.groups())
                if total == request["steps"]:
                    emit(
                        "progress", progress=0.15 + 0.65 * completed / total,
                        detail="Decoding video" if completed == total else "Generating video",
                    )
        if process.wait() != 0:
            raise RuntimeError("Wan generation failed. Check the generation log")
    if not output.is_file() or output.stat().st_size == 0:
        raise RuntimeError("Wan did not produce a video")


def generate(request):
    import imageio_ffmpeg

    with tempfile.TemporaryDirectory(dir=Path(request["output"]).parent) as stage:
        intermediate = Path(stage) / "video.avi"
        run(request, intermediate)
        emit("progress", progress=0.95, detail="Encoding video")
        result = subprocess.run(
            [
                imageio_ffmpeg.get_ffmpeg_exe(), "-nostdin", "-y",
                "-loglevel", "error", "-i", str(intermediate),
                "-an", "-c:v", "libx264", "-crf", "18",
                "-pix_fmt", "yuv420p", "-movflags", "+faststart", request["output"],
            ],
            stdout=sys.stderr, stderr=sys.stderr, check=False,
        )
        if result.returncode:
            raise RuntimeError("Could not encode Wan video. Check the generation log")
    return {
        "mime": "video/mp4", "width": request["width"],
        "height": request["height"], "duration": request["frames"] / 16,
    }
