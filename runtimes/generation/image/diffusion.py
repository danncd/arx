from events import emit


def generate(request):
    import torch
    from diffusers import AutoPipelineForText2Image, AutoPipelineForImage2Image
    from PIL import Image

    emit("progress", progress=0, detail="Loading image model")
    source = request.get("source")
    pipeline_type = AutoPipelineForImage2Image if source else AutoPipelineForText2Image
    pipeline = pipeline_type.from_pretrained(
        request["model_path"],
        torch_dtype=torch.float16,
        variant="fp16",
        local_files_only=True,
        use_safetensors=True,
    ).to("mps")
    pipeline.enable_attention_slicing()
    steps = request.get("steps", 4)

    def progress(pipe, index, timestep, values):
        emit("progress", progress=(index + 1) / steps, detail="Generating image")
        return values

    options = {
        "prompt": request["prompt"],
        "num_inference_steps": steps,
        "guidance_scale": 7.5 if request.get("runtime") == "sd" else 0.0,
        "generator": torch.Generator(device="cpu").manual_seed(request["seed"]),
        "callback_on_step_end": progress,
    }
    if source:
        options["image"] = Image.open(source).convert("RGB").resize(
            (request["width"], request["height"])
        )
        options["strength"] = 0.75
    else:
        options["width"], options["height"] = request["width"], request["height"]
    image = pipeline(**options).images[0]
    image.save(request["output"])
    return {"mime": "image/png", "width": image.width, "height": image.height}
