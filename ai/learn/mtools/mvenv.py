import os
import venv
import subprocess
from typing import List, Tuple, Dict


def open_venv(
    venv_path: str = "venv",
    prompt_user: bool = True,
    torch_index_url: str = "https://download.pytorch.org/whl/cu130",
) -> Tuple[bool, str, str, str]:
    venv_path = os.path.abspath(venv_path)
    is_windows = (os.name == "nt")

    import_to_pip: Dict[str, str] = {
        "pandas":       "pandas",
        "numpy":        "numpy",
        "chess":        "python-chess",
        "torch":        "torch",        # special-case: will install torchvision/torchaudio together if user agrees
        "sklearn":      "scikit-learn", # `from sklearn import ...`
        "tqdm":         "tqdm",
        # "onnx":         "onnx",
        # "onnxscript":   "onnxscript",
    }

    pip_packages_ordered: List[str] = [
        import_to_pip["pandas"],
        import_to_pip["numpy"],
        import_to_pip["chess"],
        import_to_pip["torch"],
        import_to_pip["sklearn"],
        import_to_pip["tqdm"],
        # import_to_pip["onnx"],
        # import_to_pip["onnxscript"],
    ]

    if not os.path.exists(venv_path):
        print(f"Creating {venv_path} ...")
        venv.EnvBuilder(with_pip=True).create(venv_path)
    else:
        print("Venv has been alredy created (use that)")

    if is_windows:
        pip_exe = os.path.join(venv_path, "Scripts", "pip.exe")
        python_in_venv = os.path.join(venv_path, "Scripts", "python.exe")
    else:
        pip_exe = os.path.join(venv_path, "bin", "pip")
        python_in_venv = os.path.join(venv_path, "bin", "python")

    if not os.path.exists(pip_exe):
        pip_exe = None

    def run_pip(args: List[str]) -> subprocess.CompletedProcess:
        if pip_exe and os.path.exists(pip_exe):
            cmd = [pip_exe] + args
        else:
            cmd = [python_in_venv, "-m", "pip"] + args
        return subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    
    def run_py(args: List[str]) -> subprocess.CompletedProcess:
        cmd = [python_in_venv] + args
        return subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)

    print("Upgrade pip/setuptools/wheel...")
    r = run_py(["-m", "pip", "install", "--upgrade", "pip", "setuptools", "wheel"])
    if r.returncode == 0:
        print("Success")
    else:
        print("Warning: pip is not upgraded")
        print(r.stdout)
        print(r.stderr)

    def is_installed(pip_name: str) -> bool:
        r = run_pip(["show", pip_name])
        return r.returncode == 0

    display_map = {
        "pandas":       "pandas",
        "numpy":        "numpy",
        "chess":        "python-chess (import: chess)",
        "torch":        "torch (+ torchvision, torchaudio)",
        "sklearn":      "scikit-learn (import: sklearn)",
        "tqdm":         "tqdm",
        # "onnx":         "onnx (+ onnxscript)",
    }

    for pip_name in pip_packages_ordered:
        import_name = None
        for k, v in import_to_pip.items():
            if v == pip_name:
                import_name = k
                break

        if import_name is None:
            display_name = pip_name
        else:
            display_name = display_map.get(import_name, pip_name)

        print(f"\nCheck pkg: {display_name} ...")
        if is_installed(pip_name):
            print(f"  → {display_name} is already installed")
            continue

        if prompt_user:
            should_install = True
        else:
            if pip_name == "torch":
                question = (
                    f"Package {display_name} is not found. Install torch, torchvision & torchaudio"
                    f"with index {torch_index_url}? (~5min) [y/N]: "
                )
            else:
                question = f"Package {display_name} is not found. Install {pip_name}? [y/N]: "
            answer = input(question).strip().lower()
            should_install = answer in ("y", "yes")

        if not should_install:
            print(f"  → Skip install {display_name}.")
            continue

        if pip_name == "torch":
            install_args = ["install", "torch", "torchvision", "torchaudio", "--index-url", torch_index_url]
        # elif pip_name == "onnx":
        #     install_args = ["install", "onnx", "onnxscript"]
        else:
            install_args = ["install", pip_name]

        print(f"  → Run: pip {' '.join(install_args)} ... (inside {venv_path})")
        r = run_pip(install_args)
        if r.returncode == 0:
            print(f"  ✓ Installed {display_name}.")
        else:
            print(f"  ✗ Error install {display_name} (code {r.returncode}).")
            print("  stdout:", r.stdout.strip())
            print("  stderr:", r.stderr.strip())
            print("\nDelete \"venv\" folder if you want to reinstall\n")
            return False, "", "", ""

    print("\nVenv is ready!")
    if is_windows:
        return True, venv_path, f"{venv_path}\\Scripts\\activate", python_in_venv
    else:
        return True, venv_path, "source {venv_path}/bin/activate", python_in_venv