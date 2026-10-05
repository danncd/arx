from events import emit


class Progress:
    def call_in_loop(self, t, config, **kwargs):
        completed = t - config.init_time_step + 1
        total = config.num_inference_steps - config.init_time_step
        emit("progress", progress=completed / max(total, 1), detail="Generating image")


def generate(request):
    import mlx.core as mx
    from mflux.models.common.config import ModelConfig

    mx.set_cache_limit(256 * 1024 * 1024)
    emit("progress", progress=0, detail="Loading image model")
    source = request.get("source")
    sources = request.get("sources") or ([source] if source else [])
    if len(sources) > 1 and request["runtime"] != "flux2":
        raise ValueError("This image model accepts only one reference")
    source = sources[0] if sources else None
    options = {}
    if request["runtime"] == "flux2":
        from mflux.models.flux2.variants import Flux2Klein, Flux2KleinEdit

        model_type = Flux2KleinEdit if source else Flux2Klein
        config = ModelConfig.flux2_klein_4b()
        if source:
            options["image_paths"] = sources
    else:
        from mflux.models.z_image import ZImage

        model_type = ZImage
        config = ModelConfig.z_image_turbo()
        if source:
            options.update(image_path=source, image_strength=0.75)
    model = model_type(model_path=request["model_path"], model_config=config, quantize=4)
    model.callbacks.register(Progress())
    emit("progress", progress=0, detail="Generating image")
    result = model.generate_image(
        seed=request["seed"],
        prompt=request["prompt"],
        width=request["width"],
        height=request["height"],
        num_inference_steps=request["steps"],
        **options,
    )
    result.image.save(request["output"])
    return {"mime": "image/png", "width": result.image.width, "height": result.image.height}
