#!/usr/bin/python
from  pathlib import Path
import os, sys
from tabulate import tabulate
import pbhelper
import orca

if sys.argv[1] == '--fullexport' or sys.argv[1] == '--deltaexport':
	work_dir = Path(sys.argv[2])
	targets = pbhelper.find_targets(work_dir)

	for target in targets:
		orca_session = orca.open_session(target.own_path)
		for pbl_path in target.get_pbl_list():
			export_folder = pbl_path.own_path.parent.joinpath('src').joinpath(pbl_path.own_path.stem)

			if export_folder.exists():
				# if pbl was not changed since last export then skip it
				if sys.argv[1] == '--deltaexport':
					if export_folder.stat().st_mtime >= pbl_path.own_path.stat().st_mtime:
						continue
				
				# all files are deleted first because they could be deleted in the library
				for source_file in export_folder.iterdir():
					source_file.unlink()
			else:
				export_folder.mkdir(parents = True, exist_ok = True)
			
			print('begin with export of {}'.format(pbl_path))
			
			orca.export_pbl(orca_session, pbl_path.own_path, export_folder)

			# set modified date of export folder to a newer value than the the one of its library
			export_folder.touch()

	print('export finished')

elif sys.argv[1] == "--libimport" or sys.argv[2] == "--libimport":
	if sys.argv[1] == "--libimport":
		#contextmenu pbl -> Import
		basePath = os.path.dirname(sys.argv[2])
		library =  os.path.basename(sys.argv[2])
		target = os.path.join(basePath, list(Path(basePath).glob('*.pbt'))[0])
		folder = os.path.join(basePath, "export_" + library[:-4])
	else:
		target = sys.argv[1]
		library = sys.argv[3]
		folder = sys.argv[4]
	
	po = orca.open_session(Path(target))
	print("setLibraryList", po.setLibraryList(orca.get_library_list(target)))
	print("setApplication", po.setApplication("inf1.pbl", "a3"))
	print("create", po.libraryCreate(library, "Created by Import"))
	print(tabulate(orca.import_pbl(po, library, folder), headers=["State", "File", "ErrorList"]))
elif sys.argv[1] == "--libexport" or sys.argv[2] == "--libexport":
	if sys.argv[1] == "--libexport":
		#contextmenu pbl -> Export
		basePath = os.path.dirname(sys.argv[2])
		library =  os.path.basename(sys.argv[2])
		target = os.path.join(basePath, list(Path(basePath).glob('*.pbt'))[0])
		folder = os.path.join(basePath, "export_" + library[:-4])
	else:
		target = sys.argv[1]
		library = sys.argv[3]
		folder = sys.argv[4]

	po = orca.open_session(Path(target))
	print("setLibraryList", po.setLibraryList(orca.get_library_list(target)))
	print("setApplication", po.setApplication("inf1.pbl", "a3"))
	print(library)
	print(folder)
	print(tabulate(orca.export_pbl(po, library, folder), headers=["File", "Type", "Size", "ModifiedDateTime", "Comment", "ObjectSize", "SourceSize", "ExportResult", "Source"]))

else:
	raise NotImplementedError
