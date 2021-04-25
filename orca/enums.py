from enum import Enum
from pathlib import Path

PBORCA_MSGBUFFER = 256 # Suggested msg buffer size
PBORCA_MAXCOMMENT = 255 # Maximum comment size

class PBArg(Enum):
	INPUT = 1
	OUTPUT = 2

class PBEncoding(Enum):
	UNICODE		= 0
	UTF8		= 1
	HEXASCII	= 2
	ANSI_DBCS	= 3

class PBSrcType(Enum):
	UNKNOWN		= None
	APPLICATION = 0
	DATAWINDOW  = 1
	FUNCTION    = 2
	MENU        = 3
	QUERY       = 4
	STRUCTURE   = 5
	USEROBJECT  = 6
	WINDOW      = 7
	PIPELINE    = 8
	PROJECT     = 9
	PROXYOBJECT = 10
	BINARY      = 11
	
	@classmethod
	def get_type(cls, source_file : Path):
		switcher = {
			'.sra' : cls.APPLICATION,
			'.srd' : cls.DATAWINDOW,
			'.srf' : cls.FUNCTION,
			'.srm' : cls.MENU,
			'.srq' : cls.QUERY,
			'.srs' : cls.STRUCTURE,
			'.sru' : cls.USEROBJECT,
			'.srw' : cls.WINDOW,
			'.srp' : cls.PIPELINE,
			'.srj' : cls.PROJECT,
			'.srpxo' : cls.PROXYOBJECT, # Deprecated, correct file ending unknown
			'.srbin' : cls.BINARY # Deprecated, correct file ending unknown
		}
		return switcher.get(source_file.suffix, cls.UNKNOWN)

	def get_file_ending(self):
		switcher = {
			PBSrcType.APPLICATION : '.sra',
			PBSrcType.DATAWINDOW : '.srd',
			PBSrcType.FUNCTION : '.srf',
			PBSrcType.MENU : '.srm',
			PBSrcType.QUERY : '.srq',
			PBSrcType.STRUCTURE : '.srs',
			PBSrcType.USEROBJECT : '.sru',
			PBSrcType.WINDOW : '.srw',
			PBSrcType.PIPELINE : '.srp',
			PBSrcType.PROJECT : '.srj',
			PBSrcType.PROXYOBJECT : '.srpxo', # Deprecated, correct file ending unknown
			PBSrcType.BINARY : '.srbin' # Deprecated, correct file ending unknown
		}
		return switcher.get(self, None)

class PBResult(Enum):
	PBORCA_OK = 0
	PBORCA_INVALIDPARMS = -1
	PBORCA_DUPOPERATION = -2
	PBORCA_OBJNOTFOUND = -3
	PBORCA_BADLIBRARY = -4
	PBORCA_LIBLISTNOTSET = -5
	PBORCA_LIBNOTINLIST = -6
	PBORCA_LIBIOERROR = -7
	PBORCA_OBJEXISTS = -8
	PBORCA_INVALIDNAME = -9
	PBORCA_BUFFERTOOSMALL = -10
	PBORCA_COMPERROR = -11
	PBORCA_LINKERROR = -12
	PBORCA_CURRAPPLNOTSET = -13
	PBORCA_OBJHASNOANCS = -14
	PBORCA_OBJHASNOREFS = -15
	PBORCA_PBDCOUNTERROR = -16
	PBORCA_PBDCREATERRORPBD = -17
	PBORCA_CHECKOUTERROR = -18
	PBORCA_CBCREATEERROR = -19
	PBORCA_CBINITERROR = -20
	PBORCA_CBBUILDERROR = -21
	PBORCA_SCCFAILURE = -22
	PBORCA_REGREADERROR = -23
	PBORCA_SCCLOADDLLFAILED = -24
	PBORCA_SCCINITFAILED = -25
	PBORCA_OPENPROJFAILED = -26
	PBORCA_TARGETNOTFOUND = -27
	PBORCA_TARGETREADERR = -28
	PBORCA_GETINTERFACEERROR = -29
	PBORCA_IMPORTONLY_REQ = -30
	PBORCA_GETCONNECT_REQSCC = -31
	PBORCA_PBCFILE_REQSCC = -32

	def as_string(self):
		return PBResult.to_string(self.value)

	@classmethod
	def to_string(cls, value):
		switcher = {
			cls.PBORCA_OK : 'Operation successful',
			cls.PBORCA_INVALIDPARMS : 'Invalid parameter list',
			cls.PBORCA_DUPOPERATION : 'Duplicate operation',
			cls.PBORCA_OBJNOTFOUND : 'Object not found',
			cls.PBORCA_BADLIBRARY : 'Bad library name',
			cls.PBORCA_LIBLISTNOTSET : 'Library list not set',
			cls.PBORCA_LIBNOTINLIST : 'Library not in library list',
			cls.PBORCA_LIBIOERROR : 'Library I/O error',
			cls.PBORCA_OBJEXISTS : 'Object exists',
			cls.PBORCA_INVALIDNAME : 'Invalid name',
			cls.PBORCA_BUFFERTOOSMALL : 'Buffer size is too small',
			cls.PBORCA_COMPERROR : 'Compile error',
			cls.PBORCA_LINKERROR : 'Link error',
			cls.PBORCA_CURRAPPLNOTSET : 'Current application not set',
			cls.PBORCA_OBJHASNOANCS : 'Object has no ancestors',
			cls.PBORCA_OBJHASNOREFS : 'Object has no references',
			cls.PBORCA_PBDCOUNTERROR : 'Invalid # of PBDs',
			cls.PBORCA_PBDCREATERRORPBD : 'create error',
			cls.PBORCA_CHECKOUTERROR : 'Source Management error (obsolete)',
			cls.PBORCA_CBCREATEERROR : 'Could not instantiate ComponentBuilder class',
			cls.PBORCA_CBINITERROR : 'Component builder Init method failed',
			cls.PBORCA_CBBUILDERROR : 'Component builder BuildProject method failed',
			cls.PBORCA_SCCFAILURE : 'Could not connect to source control',
			cls.PBORCA_REGREADERROR : 'Could not read registry',
			cls.PBORCA_SCCLOADDLLFAILED : 'Could not load DLL',
			cls.PBORCA_SCCINITFAILED : 'Could not initialize SCC connection',
			cls.PBORCA_OPENPROJFAILED : 'Could not open SCC project',
			cls.PBORCA_TARGETNOTFOUND : 'Target File not found',
			cls.PBORCA_TARGETREADERR : 'Unable to read Target File',
			cls.PBORCA_GETINTERFACEERROR : 'Unable to access SCC interface',
			cls.PBORCA_IMPORTONLY_REQ : 'Scc connect offline requires IMPORTONLY refresh option',
			cls.PBORCA_GETCONNECT_REQSCC : 'connect offline requires GetConnectProperties with Exclude_Checkout',
			cls.PBORCA_PBCFILE_REQSCC : 'connect offline with Exclude_Checkout requires PBC file',
		}
		return switcher.get(PBResult(value), 'Unknown')
