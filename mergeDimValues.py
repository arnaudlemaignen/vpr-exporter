#!/usr/bin/env python3
import sys
from ruamel.yaml import YAML
import copy

def deep_merge(dict1, dict2):
    """Recursively merge dict2 into dict1"""
    result = copy.deepcopy(dict1)
    for key, value in dict2.items():
        if (
            key in result 
            and isinstance(result[key], dict) 
            and isinstance(value, dict)
        ):
            result[key] = deep_merge(result[key], value)
        else:
            result[key] = copy.deepcopy(value)
    return result

def merge_yaml_files(origin_path, override_path, output_path):
    yaml = YAML()
    try:
        with open(origin_path, 'r') as f1, open(override_path, 'r') as f2:
            origin = yaml.load(f1) or {}
            override = yaml.load(f2) or {}

        merged = deep_merge(origin, override)

        with open(output_path, 'w') as f_out:
            yaml.dump(merged, f_out)

        print(f"Merged YAML written to: {output_path}")
    except Exception as e:
        print(f"Error: {e}")

if __name__ == "__main__":
    if len(sys.argv) != 4:
        print("Usage: python merge_yaml.py <origin.yaml> <override.yaml> <output.yaml>")
        sys.exit(1)

    merge_yaml_files(sys.argv[1], sys.argv[2], sys.argv[3])
