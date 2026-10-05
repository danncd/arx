import subprocess
from events import emit


def generate(request):
    import imageio_ffmpeg

    emit("progress", progress=0, detail="Combining video and narration")
    command = [
        imageio_ffmpeg.get_ffmpeg_exe(),
        "-nostdin", "-y", "-loglevel", "error",
        "-i", request["source"],
        "-i", request["audio"],
        "-map", "0:v:0", "-map", "1:a:0",
        "-c:v", "copy", "-c:a", "aac",
        "-shortest", "-movflags", "+faststart", request["output"],
    ]
    result = subprocess.run(command, capture_output=True, text=True, check=False)
    if result.returncode:
        raise ValueError(result.stderr[-2000:] or "Media processing failed")
    return {"mime": "video/mp4"}
