import os
import tempfile
from pathlib import Path
from events import emit


def generate(request):
    if request.get("runtime") == "qwen3-tts":
        return synthesize(request)
    import espeakng_loader

    with tempfile.TemporaryDirectory(prefix="arx-voice-", dir="/tmp") as temporary:
        Path(temporary, "espeak-ng-data").symlink_to(espeakng_loader.get_data_path(), target_is_directory=True)
        os.environ["ESPEAK_DATA_PATH"] = temporary
        return synthesize(request)


def synthesize(request):
    import numpy as np
    import soundfile as sf
    import espeakng_loader
    from phonemizer.backend.espeak.wrapper import EspeakWrapper

    os.environ["PHONEMIZER_ESPEAK_LIBRARY"] = espeakng_loader.get_library_path()
    os.environ["PHONEMIZER_ESPEAK_DATA_PATH"] = espeakng_loader.get_data_path()
    EspeakWrapper.set_library(espeakng_loader.get_library_path())
    EspeakWrapper.set_data_path(espeakng_loader.get_data_path())
    from mlx_audio.tts.utils import load_model

    emit("progress", progress=0, detail="Loading speech model")
    model = load_model(request["model_path"])
    voice = request.get("voice", "af_heart")
    if request.get("runtime") not in ("kitten", "qwen3-tts"):
        voice = str(Path(request["model_path"]) / "voices" / (voice + ".safetensors"))
    parts = []
    sample_rate = 24000
    for result in model.generate(
        text=request["prompt"],
        voice=voice,
        speed=request.get("speed", 1.0),
        lang_code="auto" if request.get("runtime") == "qwen3-tts" else "a",
    ):
        parts.append(np.asarray(result.audio).reshape(-1))
        sample_rate = result.sample_rate
        emit("progress", progress=0, detail="Generating speech")
    if not parts:
        raise ValueError("Speech model returned no audio")
    audio = np.concatenate(parts)
    sf.write(request["output"], audio, sample_rate, subtype="PCM_16")
    return {"mime": "audio/wav", "duration": len(audio) / sample_rate}
