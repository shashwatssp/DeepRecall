# Models Folder

This folder stores AI model files.

## Whisper Models (STT)

Download from: https://huggingface.co/ggerganov/whisper.cpp/tree/main

### Recommended Models:

**Base (158 MB) - Good for most use cases:**
```bash
curl -L -o models/ggml-base.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin
Small (466 MB) - Better accuracy:

bash
curl -L -o models/ggml-small.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin
Medium (1.5 GB) - High accuracy:

bash
curl -L -o models/ggml-medium.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin
English-only Models (Faster):
bash
curl -L -o models/ggml-base.en.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin
Configuration
Update config/config.yaml:

yaml
stt:
  model_path: "./models/ggml-base.bin"
Storage
Models are large binary files

Not tracked in Git (see .gitignore)