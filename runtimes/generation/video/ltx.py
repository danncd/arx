from events import emit


def generate(request):
    import torch
    from diffusers import LTXPipeline, LTXImageToVideoPipeline
    from diffusers.utils import export_to_video
    from PIL import Image

    emit("progress", progress=0, detail="Loading video model")
    source = request.get("source")
    pipeline_type = LTXImageToVideoPipeline if source else LTXPipeline
    pipeline = pipeline_type.from_pretrained(
        request["model_path"],
        torch_dtype=torch.bfloat16,
        local_files_only=True,
        use_safetensors=True,
    )
    pipeline.enable_sequential_cpu_offload(device="mps")
    pipeline.vae.enable_tiling()
    steps = request.get("steps", 30)
    torch.manual_seed(request["seed"])
    torch.mps.manual_seed(request["seed"])

    def progress(pipe, index, timestep, values):
        emit("progress", progress=(index + 1) / steps, detail="Generating video")
        return values

    options = {
        "prompt": request["prompt"],
        "negative_prompt": "blurry, distorted, low quality",
        "width": request["width"],
        "height": request["height"],
        "num_frames": request["frames"],
        "frame_rate": 8,
        "num_inference_steps": steps,
        "guidance_scale": 3.0,
        "callback_on_step_end": progress,
    }
    if source:
        options["image"] = Image.open(source).convert("RGB").resize(
            (request["width"], request["height"])
        )
    frames = pipeline(**options).frames[0]
    emit("progress", progress=1, detail="Encoding video")
    export_to_video(frames, request["output"], fps=8)
    return {
        "mime": "video/mp4",
        "width": request["width"],
        "height": request["height"],
        "duration": len(frames) / 8,
    }
