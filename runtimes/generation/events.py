import json


def emit(event, **values):
    print(json.dumps({"event": event, **values}), flush=True)
