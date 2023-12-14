$PBExportHeader$inf1_u_extractorobject.sru
forward
global type inf1_u_extractorobject from extractorobject
end type
end forward

global type inf1_u_extractorobject from extractorobject
end type
global inf1_u_extractorobject inf1_u_extractorobject

on inf1_u_extractorobject.create
call super::create
TriggerEvent( this, "constructor" )
end on

on inf1_u_extractorobject.destroy
TriggerEvent( this, "destructor" )
call super::destroy
end on

