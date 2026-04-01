# inspect_helper.py
import sys
import importlib
import inspect
import json

def get_class_signature(target_path):
    try:
        parts = target_path.rsplit('.', 1)
        if len(parts) != 2:
            return {"error": "Invalid target path. Use format: module.ClassName"}

        module_name, class_name = parts
        module = importlib.import_module(module_name)
        cls = getattr(module, class_name)

        sig = inspect.signature(cls.__init__)
        params = []
        for name, param in sig.parameters.items():
            if name == 'self' or name == 'args' or name == 'kwargs':
                continue

            has_default = param.default is not inspect.Parameter.empty
            default_val = param.default if has_default else None

            if has_default and not isinstance(default_val, (int, float, str, bool, type(None))):
                default_val = str(default_val)

            # 型ヒントの取得
            type_hint = ""
            if param.annotation is not inspect.Parameter.empty:
                if hasattr(param.annotation, '__name__'):
                    type_hint = param.annotation.__name__
                else:
                    type_hint = str(param.annotation).replace('typing.', '')

            params.append({
                "name": name,
                "has_default": has_default,
                "default": default_val,
                'type': type_hint
            })

        return {"target": target_path, "params": params}
    except Exception as e:
        return {"error": str(e)}

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No target provided"}))
        sys.exit(1)

    target = sys.argv[1]
    print(json.dumps(get_class_signature(target)))