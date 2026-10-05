import sys
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from video import wan


class Process:
    def __init__(self, lines, code):
        self.stdout = iter(lines)
        self.code = code

    def __enter__(self):
        return self

    def __exit__(self, *args):
        pass

    def wait(self):
        return self.code


class WanTests(unittest.TestCase):
    def setUp(self):
        self.request = {
            "model_path": "/tmp/model path", "prompt": "apple; unchanged",
            "steps": 30, "width": 384, "height": 256, "frames": 25, "seed": 42,
        }

    def test_progress_ignores_weight_transfers(self):
        lines = [
            "|####| 30/30 - 1.2GB/s\n",
            "|====| 15/30 - 11.2s/it\n",
            "|====| 30/30 - 11.2s/it\n",
        ]
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "video.avi"
            output.write_bytes(b"video")
            with patch.object(wan.subprocess, "Popen", return_value=Process(lines, 0)):
                with patch.object(wan, "emit") as emit:
                    wan.run(self.request, output)
            events = [call.kwargs for call in emit.call_args_list]
            self.assertEqual([event["progress"] for event in events], [0, 0.475, 0.8])
            self.assertEqual(events[-1]["detail"], "Decoding video")

    def test_failures_never_report_success(self):
        with tempfile.TemporaryDirectory() as directory:
            for code in (0, 1):
                with self.subTest(code=code):
                    with patch.object(wan.subprocess, "Popen", return_value=Process([], code)):
                        with patch.object(wan, "emit"):
                            with self.assertRaises(RuntimeError):
                                wan.run(self.request, Path(directory) / "missing.avi")

    def test_prompt_is_one_argument(self):
        arguments = wan.command(self.request, Path("/tmp/output.avi"))
        self.assertEqual(arguments[arguments.index("-p") + 1], self.request["prompt"])
        self.assertIn("/tmp/model path/wan2.1_t2v_1.3B_fp16.safetensors", arguments)


if __name__ == "__main__":
    unittest.main()
