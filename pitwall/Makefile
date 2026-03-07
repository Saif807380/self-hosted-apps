VENV := venv
PYTHON := $(VENV)/bin/python
PIP := $(VENV)/bin/pip

.PHONY: setup run test clean

setup:
	/usr/bin/python3 -m venv $(VENV)
	source $(VENV)/bin/activate
	$(PIP) install --upgrade pip
	$(PIP) install -r requirements.txt
	$(VENV)/bin/playwright install chromium
	@echo "\nSetup complete. Activate with: source $(VENV)/bin/activate"

run:
	$(PYTHON) main.py

test:
	$(PYTHON) -m pytest tests/ -v

clean:
	rm -rf $(VENV) __pycache__ */__pycache__ */*/__pycache__
