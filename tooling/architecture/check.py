import ast
import io
import sys
import tokenize
from pathlib import Path

root = Path(sys.argv[1])
failures = []
for path in root.rglob('*.py'):
    if any(part in {'node_modules', '.build', 'dist', '.git', '__pycache__'} for part in path.relative_to(root).parts):
        continue
    source = path.read_text()
    tree = ast.parse(source, filename=str(path))
    if any(token.type == tokenize.COMMENT for token in tokenize.generate_tokens(io.StringIO(source).readline)):
        failures.append(f'Source comments: {path.relative_to(root)}')
    for node in ast.walk(tree):
        if isinstance(node, (ast.Module, ast.ClassDef, ast.FunctionDef, ast.AsyncFunctionDef)) and ast.get_docstring(node):
            failures.append(f'Docstring outside docs: {path.relative_to(root)}')
if failures:
    raise SystemExit('\n'.join(failures))
print('Python syntax/comment policy passed')
