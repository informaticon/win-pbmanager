$PBExportHeader$inf1_u_dotnetassembly.sru
forward
global type inf1_u_dotnetassembly from dotnetassembly
end type
end forward

global type inf1_u_dotnetassembly from dotnetassembly
end type
global inf1_u_dotnetassembly inf1_u_dotnetassembly

on inf1_u_dotnetassembly.create
call super::create
TriggerEvent( this, "constructor" )
end on

on inf1_u_dotnetassembly.destroy
TriggerEvent( this, "destructor" )
call super::destroy
end on

