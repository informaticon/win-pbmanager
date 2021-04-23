from pathlib import Path
import pbfiles



for target in pbfiles.find_targets(Path('C:\\a3\\a_kaeppeli')):
	print(target.name)
