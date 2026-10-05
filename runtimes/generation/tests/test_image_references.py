import sys
import unittest
from pathlib import Path
from types import ModuleType
from unittest.mock import MagicMock, patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from image import mlx as mlx_image


class ImageReferencesTests(unittest.TestCase):
    def run_image(self, runtime, **sources):
        model = MagicMock()
        config = MagicMock()
        modules = {}
        for name in ["mlx", "mlx.core", "mflux", "mflux.models", "mflux.models.common", "mflux.models.common.config", "mflux.models.flux2", "mflux.models.flux2.variants", "mflux.models.z_image"]:
            modules[name] = ModuleType(name)
        modules["mlx"].core = modules["mlx.core"]
        modules["mlx.core"].set_cache_limit = MagicMock()
        modules["mflux.models.common.config"].ModelConfig = config
        modules["mflux.models.flux2.variants"].Flux2KleinEdit = model
        modules["mflux.models.flux2.variants"].Flux2Klein = model
        modules["mflux.models.z_image"].ZImage = model
        request = dict(runtime=runtime, model_path="model", seed=1, prompt="Keep the characters", width=512, height=512, steps=4, output="output.png", **sources)
        with patch.dict(sys.modules, modules), patch.object(mlx_image, "emit"):
            mlx_image.generate(request)
        return model.return_value.generate_image.call_args.kwargs

    def test_flux_receives_all_references_in_order(self):
        options = self.run_image("flux2", sources=["character.png", "strip.png"])
        self.assertEqual(options["image_paths"], ["character.png", "strip.png"])

    def test_legacy_source_and_z_image_still_work(self):
        self.assertEqual(self.run_image("flux2", source="strip.png")["image_paths"], ["strip.png"])
        self.assertEqual(self.run_image("z-image", source="strip.png")["image_path"], "strip.png")

    def test_z_image_rejects_multiple_references(self):
        with self.assertRaisesRegex(ValueError, "only one reference"):
            self.run_image("z-image", sources=["one.png", "two.png"])


if __name__ == "__main__":
    unittest.main()
