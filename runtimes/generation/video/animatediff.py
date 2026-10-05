from pathlib import Path
from events import emit


def generate(request):
    import torch
    from diffusers import AnimateDiffPipeline, MotionAdapter, EulerDiscreteScheduler
    from diffusers.utils import export_to_video
    from safetensors.torch import load_file

    emit("progress", progress=0, detail="Loading animation model")
    directory = Path(request["model_path"])
    adapter = MotionAdapter()
    adapter.load_state_dict(load_file(str(directory / "animatediff_lightning_4step_diffusers.safetensors")))
    pipeline = AnimateDiffPipeline.from_pretrained(
        str(directory / "base"),
        motion_adapter=adapter,
        torch_dtype=torch.float16,
        variant="fp16",
        local_files_only=True,
        use_safetensors=True,
    )
    pipeline.scheduler = EulerDiscreteScheduler.from_config(
        pipeline.scheduler.config, timestep_spacing="trailing", beta_schedule="linear"
    )
    pipeline.enable_model_cpu_offload(device="mps")
    pipeline.enable_vae_slicing()
    pipeline.unet.enable_forward_chunking(chunk_size=1, dim=1)
    steps = request.get("steps", 4)

    def progress(pipe, index, timestep, values):
        emit("progress", progress=(index + 1) / steps, detail="Generating video")
        return values

    frames = pipeline(
        prompt=request["prompt"],
        width=request["width"],
        height=request["height"],
        num_frames=request["frames"],
        guidance_scale=1.0,
        num_inference_steps=steps,
        generator=torch.Generator(device="cpu").manual_seed(request["seed"]),
        callback_on_step_end=progress,
    ).frames[0]
    emit("progress", progress=1, detail="Encoding video")
    export_to_video(frames, request["output"], fps=8)
    return {
        "mime": "video/mp4",
        "width": request["width"],
        "height": request["height"],
        "duration": len(frames) / 8,
    }
