SHELL=cmd

version = 0.1.0

build:
	python -m nuitka --standalone pborca/pborca.py --company-name="Informaticon AG" --product-name="PB Orca Interface" --product-version="$(version)" --windows-icon-from-ico=assets/logo.ico --output-dir="deploy/build/" --warn-unusual-code --warn-implicit-exceptions