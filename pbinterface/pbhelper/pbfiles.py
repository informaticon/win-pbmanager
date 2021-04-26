from pathlib import Path
import re

_re_app_name = re.compile('appname "(.*)";')
_re_app_lib = re.compile('applib "(.*)";')
_re_lib_list = re.compile('LibList "(.*)";')

class Pbl:
	def __init__(self, own_path : Path):
		self.own_path = own_path
	
	def __str__(self):
		return 'PBL: {} ({})'.format(self.own_path.name, self.own_path)
	
	def get_path(self):
		return self.own_path
		
	@classmethod
	def from_base_folder(cls, base_folder : Path, pbl_path : str):
		return Pbl(base_folder.joinpath(pbl_path))
		

class Pbt:
	def __init__(self, own_path : Path, pbl_list : list[Pbl] = None, app_name : str = None, app_lib : str = None):
		self.own_path = own_path
		self.app_name = app_name
		self.app_lib = app_lib
		if pbl_list == None:
			self.pbl_list = []
		else:
			self.pbl_list = pbl_list
			
	def __str__(self):
		return 'PBT: {} ({})'.format(self.app_name, self.own_path)
	
	def add_pbl(self, pbl : Pbl):
		self.pbl_list.append(pbl)
	
	def get_pbl_list(self):
		return self.pbl_list

	def get_folder(self):
		return self.own_path.parent
	
	def get_own_path(self):
		return self.own_path
		
	def get_app_name(self):
		return self.app_name

	def get_app_pbl_path(self):
		return self.app_lib
	
	@classmethod
	def parse(cls, pbt_path : Path):
		pbt = Pbt(pbt_path)
		with open(pbt_path, 'r') as fp:
			for _, line in enumerate(fp):
				m = _re_app_name.match(line)
				if m:
					pbt.app_name = m.group(1)
				
				m = _re_app_lib.match(line)
				if m:
					pbt.app_lib = pbt_path.parent.joinpath(m.group(1))
			
				m = _re_lib_list.match(line)
				if m:
					for lib in m.group(1).split(';'):
						pbt.add_pbl(Pbl.from_base_folder(pbt.get_folder(), lib))


		return pbt

def find_targets(search_path : Path) -> list[Pbt]:
	return list(map(lambda path: Pbt.parse(Path(path)), search_path.glob('**/*.pbt')))