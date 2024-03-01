$PBExportHeader$inf1_u_pdfpage.sru
forward
global type inf1_u_pdfpage from pdfpage
end type
end forward

global type inf1_u_pdfpage from pdfpage
end type
global inf1_u_pdfpage inf1_u_pdfpage

on inf1_u_pdfpage.create
call super::create
TriggerEvent( this, "constructor" )
end on

on inf1_u_pdfpage.destroy
TriggerEvent( this, "destructor" )
call super::destroy
end on

