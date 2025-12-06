; DeeMusic NSIS Installer Script
; Creates a Windows installer for DeeMusic

!define PRODUCT_NAME "DeeMusic"
!define PRODUCT_VERSION "2.1.3"
!define PRODUCT_PUBLISHER "DeeMusic Team"
!define PRODUCT_WEB_SITE "https://github.com/yourusername/deemusic"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
!define PRODUCT_UNINST_ROOT_KEY "HKLM"

; Modern UI
!include "MUI2.nsh"

; General
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "..\scripts\build\DeeMusic-Setup-v${PRODUCT_VERSION}.exe"
InstallDir "$PROGRAMFILES64\DeeMusic"
InstallDirRegKey HKLM "Software\${PRODUCT_NAME}" "InstallDir"
RequestExecutionLevel admin
ShowInstDetails show
ShowUnInstDetails show

; Interface Settings
!define MUI_ABORTWARNING
!define MUI_ICON "${NSISDIR}\Contrib\Graphics\Icons\modern-install.ico"
!define MUI_UNICON "${NSISDIR}\Contrib\Graphics\Icons\modern-uninstall.ico"

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\DeeMusic.Desktop.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Launch DeeMusic"
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Language
!insertmacro MUI_LANGUAGE "English"

; Version Info
VIProductVersion "${PRODUCT_VERSION}.0"
VIAddVersionKey "ProductName" "${PRODUCT_NAME}"
VIAddVersionKey "ProductVersion" "${PRODUCT_VERSION}"
VIAddVersionKey "CompanyName" "${PRODUCT_PUBLISHER}"
VIAddVersionKey "FileVersion" "${PRODUCT_VERSION}"
VIAddVersionKey "FileDescription" "${PRODUCT_NAME} Installer"
VIAddVersionKey "LegalCopyright" "© ${PRODUCT_PUBLISHER}"

Section "MainSection" SEC01
  SetOutPath "$INSTDIR"
  SetOverwrite on
  
  ; Kill any running instances before updating
  nsExec::ExecToLog 'taskkill /F /IM DeeMusic.Desktop.exe'
  Sleep 500
  
  ; Copy all files from publish directory
  File /r "..\DeeMusic.Desktop\bin\Release\net8.0-windows\win-x64\publish\*"
  
  ; Create shortcuts
  CreateDirectory "$SMPROGRAMS\DeeMusic"
  CreateShortCut "$SMPROGRAMS\DeeMusic\DeeMusic.lnk" "$INSTDIR\DeeMusic.Desktop.exe"
  CreateShortCut "$SMPROGRAMS\DeeMusic\Uninstall.lnk" "$INSTDIR\uninst.exe"
  CreateShortCut "$DESKTOP\DeeMusic.lnk" "$INSTDIR\DeeMusic.Desktop.exe"
  
  ; Write registry keys
  WriteRegStr HKLM "Software\${PRODUCT_NAME}" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "Software\${PRODUCT_NAME}" "Version" "${PRODUCT_VERSION}"
  
  ; Write uninstall information
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninst.exe"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\DeeMusic.Desktop.exe"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  
  ; Create uninstaller
  WriteUninstaller "$INSTDIR\uninst.exe"
  
  ; Refresh Windows Explorer cache to update taskbar tooltips and file properties
  ; This ensures the correct version is displayed immediately
  DetailPrint "Refreshing Windows Explorer cache..."
  nsExec::ExecToLog 'ie4uinit.exe -show'
  
  ; Notify Windows that file associations have changed
  System::Call 'shell32::SHChangeNotify(i 0x8000000, i 0, i 0, i 0)'
SectionEnd

Section "Uninstall"
  ; Kill any running instances
  nsExec::ExecToLog 'taskkill /F /IM DeeMusic.Desktop.exe'
  Sleep 1000
  
  ; Remove files
  RMDir /r "$INSTDIR"
  
  ; Remove shortcuts
  Delete "$DESKTOP\DeeMusic.lnk"
  RMDir /r "$SMPROGRAMS\DeeMusic"
  
  ; Remove registry keys
  DeleteRegKey ${PRODUCT_UNINST_ROOT_KEY} "${PRODUCT_UNINST_KEY}"
  DeleteRegKey HKLM "Software\${PRODUCT_NAME}"
  
  ; Remove user data (optional - ask user)
  MessageBox MB_YESNO "Do you want to remove all user data and settings?" IDNO skip_userdata
    RMDir /r "$APPDATA\DeeMusicV2"
  skip_userdata:
SectionEnd
