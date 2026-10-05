import shutil
import sys
import tarfile
from pathlib import Path, PurePosixPath


def install(archive, destination):
    destination.mkdir(parents=True, exist_ok=True)
    with tarfile.open(archive, "r:gz") as source:
        for entry in source:
            relative = PurePosixPath(*PurePosixPath(entry.name).parts[1:])
            name = str(relative)
            selected = (
                name.startswith((
                    "fastvideo/mlx_runtime/",
                    "fastvideo/models/schedulers/",
                    "fastvideo/logging_utils/",
                )) and name.endswith(".py")
                or name in (
                    "fastvideo/logger.py",
                    "fastvideo/envs.py",
                    "fastvideo/layers/custom_op.py",
                    "fastvideo/layers/rotary_embedding.py",
                )
                or name == "examples/inference/basic/mlx_wan_prompt_to_video.py"
                or name in ("LICENSE", "NOTICE")
            )
            if not selected or not entry.isfile() or ".." in relative.parts:
                continue
            target = destination.joinpath(*relative.parts)
            target.parent.mkdir(parents=True, exist_ok=True)
            with source.extractfile(entry) as incoming, target.open("wb") as output:
                shutil.copyfileobj(incoming, output)

    entrypoint = destination / "examples/inference/basic/mlx_wan_prompt_to_video.py"
    update(entrypoint, 'padding="max_length"', 'padding="longest"')
    update(
        entrypoint,
        "max_sequence_length=args.max_sequence_length,\n        device_arg=args.torch_device,",
        'max_sequence_length=args.max_sequence_length,\n        device_arg="cpu",',
    )
    rotary = destination / "fastvideo/layers/rotary_embedding.py"
    dependency = "from fastvideo.distributed.parallel_state import get_sp_group"
    update(rotary, dependency + "\n", "")
    update(rotary, "    if do_sp_sharding:\n", "    if do_sp_sharding:\n        " + dependency + "\n")
    (destination / "ARX-NOTICE").write_text(
        "FastVideo dd35763ad6ba8db441fdccbe05f66fba797b8931, Apache-2.0.\n"
        "Arx changes: encode unpadded text on CPU, retain the padded embedding output; "
        "defer the distributed rotary import until sharding is requested.\n"
    )


def update(path, old, new):
    text = path.read_text()
    if text.count(old) != 1:
        raise ValueError("Unsupported FastMetal source: " + str(path))
    path.write_text(text.replace(old, new))


if __name__ == "__main__":
    install(Path(sys.argv[1]), Path(sys.argv[2]))
