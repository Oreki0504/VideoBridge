#ifndef PayloadDir
  #error PayloadDir must point to the prepared installer payload
#endif
#ifndef AppVersion
  #error AppVersion is required
#endif
#ifndef HostFile
  #error HostFile is required
#endif
[Setup]
AppId={{8A8DD097-86D8-4ED0-9C18-A83A6F23DA15}
AppName=VideoBridge
AppVersion={#AppVersion}
AppPublisher=VideoBridge
DefaultDirName={localappdata}\VideoBridge
DefaultGroupName=VideoBridge
DisableDirPage=yes
DisableProgramGroupPage=yes
AllowNoIcons=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
WizardStyle=modern
Compression=lzma2
SolidCompression=yes
OutputDir=..\dist
OutputBaseFilename=VideoBridge-{#AppVersion}-Setup
UninstallDisplayName=VideoBridge
UninstallDisplayIcon={app}\unins000.exe
CloseApplications=no
RestartApplications=no
SetupMutex=VideoBridgeSetup
UsePreviousTasks=yes
DisableWelcomePage=no
ShowLanguageDialog=auto
LanguageDetectionMethod=uilanguage

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "chinesesimp"; MessagesFile: "compiler:Languages\ChineseSimplified.isl"

[Messages]
english.WelcomeLabel2=This wizard installs the VideoBridge browser extension and local download tools.%n%nProgram files are stored in your user account's VideoBridge folder. No administrator privileges are required. Existing download folders and tool settings are preserved.%n%nAfter installation, load the extension once in Chrome or Edge.
english.FinishedHeadingLabel=VideoBridge is installed
english.FinishedLabel=Follow the installation guide to load the extension from its installed folder.%n%nIf you previously loaded an unpacked copy from another folder, remove that old extension first. Downloaded files and local settings are preserved.
chinesesimp.WelcomeLabel2=此向导将安装 VideoBridge 的浏览器扩展和本地下载程序。%n%n所有程序组件集中保存在当前用户的 VideoBridge 目录，不需要管理员权限。已有下载目录和工具配置会保留。%n%n安装后只需在 Chrome / Edge 中手动加载扩展一次。
chinesesimp.FinishedHeadingLabel=VideoBridge 已安装
chinesesimp.FinishedLabel=请查看安装说明，在浏览器中加载固定目录里的扩展。%n%n如果之前使用过解压版，请移除旧扩展，再加载新目录。下载文件和本地配置会保留。

[CustomMessages]
english.ConnectChrome=Connect Chrome
english.ConnectEdge=Connect Microsoft Edge
english.InstallGuide=Installation Guide
english.ExtensionFolder=Extension Folder
english.Uninstall=Uninstall VideoBridge
english.ViewGuide=View the browser extension installation guide
english.OpenFolder=Open the extension folder
english.SelectBrowser=Select at least one browser to connect.
english.Configuring=Configuring browser connections and checking tools. First-time downloads may take a few minutes...
english.Checking=Browser connections configured. Checking local tools...
english.Installing=Downloading, verifying and installing missing tools. This may take a few minutes...
english.Ready=Local configuration is complete.
english.IncompleteHeading=Local setup needs attention
english.Incomplete=Program files are installed, but local configuration or tool installation did not finish. Check your connection and run Setup again, or click "Check & Install Dependencies" in the extension.
english.IncompleteDialog=Local configuration or tool installation did not finish. Program files have been kept. Run Setup again to continue.
chinesesimp.ConnectChrome=连接 Chrome
chinesesimp.ConnectEdge=连接 Microsoft Edge
chinesesimp.InstallGuide=安装说明
chinesesimp.ExtensionFolder=扩展文件夹
chinesesimp.Uninstall=卸载 VideoBridge
chinesesimp.ViewGuide=查看浏览器扩展安装说明
chinesesimp.OpenFolder=打开扩展文件夹
chinesesimp.SelectBrowser=请至少选择一个要连接的浏览器。
chinesesimp.Configuring=正在配置浏览器连接并检查依赖；首次下载可能需要几分钟…
chinesesimp.Checking=浏览器连接已配置，正在检查本地依赖…
chinesesimp.Installing=正在下载、校验并安装缺失工具，可能需要几分钟…
chinesesimp.Ready=本地配置完成。
chinesesimp.IncompleteHeading=还需完成本地配置
chinesesimp.Incomplete=程序文件已安装，但本地配置或依赖安装未完成。请检查网络后重新运行安装程序，或在扩展中点击“检测并补齐依赖”。
chinesesimp.IncompleteDialog=本地配置或依赖安装未完成。程序文件已保留，可重新运行安装程序继续。

[Tasks]
Name: "chrome"; Description: "{cm:ConnectChrome}"; Flags: checkedonce
Name: "edge"; Description: "{cm:ConnectEdge}"; Flags: unchecked

[Files]
Source: "{#PayloadDir}\extension\*"; DestDir: "{app}\extension"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#PayloadDir}\host\{#HostFile}"; DestDir: "{app}\host"; Flags: ignoreversion onlyifdoesntexist
Source: "{#PayloadDir}\host\build-info.json"; DestDir: "{app}\host"; Flags: ignoreversion
Source: "{#PayloadDir}\setup\*"; DestDir: "{app}\setup"; Flags: ignoreversion

[Icons]
Name: "{group}\{cm:InstallGuide}"; Filename: "{app}\setup\welcome.html"
Name: "{group}\{cm:ExtensionFolder}"; Filename: "{win}\explorer.exe"; Parameters: """{app}\extension"""
Name: "{group}\{cm:Uninstall}"; Filename: "{uninstallexe}"

[InstallDelete]
Type: files; Name: "{group}\安装说明.lnk"
Type: files; Name: "{group}\扩展文件夹.lnk"
Type: files; Name: "{group}\卸载 VideoBridge.lnk"
Type: files; Name: "{group}\Installation Guide.lnk"
Type: files; Name: "{group}\Extension Folder.lnk"
Type: files; Name: "{group}\Uninstall VideoBridge.lnk"

[Run]
Filename: "{app}\setup\welcome.html"; Description: "{cm:ViewGuide}"; Flags: shellexec postinstall skipifsilent nowait
Filename: "{win}\explorer.exe"; Parameters: """{app}\extension"""; Description: "{cm:OpenFolder}"; Flags: postinstall skipifsilent nowait unchecked

[UninstallRun]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -NonInteractive -ExecutionPolicy Bypass -File ""{app}\setup\Unregister-Installation.ps1"" -Root ""{app}"""; Flags: runhidden waituntilterminated; RunOnceId: "UnregisterNativeHost"

[Code]
var ConfigurationFailed: Boolean;

function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
  if (CurPageID = wpSelectTasks) and not (WizardIsTaskSelected('chrome') or WizardIsTaskSelected('edge')) then begin
    MsgBox(CustomMessage('SelectBrowser'), mbError, MB_OK);
    Result := False;
  end;
end;

procedure SetupLog(const S: String; const Error, FirstLine: Boolean);
begin
  Log(S);
  if S = 'VB_STATUS:checking' then WizardForm.StatusLabel.Caption := CustomMessage('Checking');
  if S = 'VB_STATUS:installing' then WizardForm.StatusLabel.Caption := CustomMessage('Installing');
  if S = 'VB_STATUS:ready' then WizardForm.StatusLabel.Caption := CustomMessage('Ready');
end;

procedure CurStepChanged(CurStep: TSetupStep);
var Browser, Params: String; ExitCode: Integer;
begin
  if CurStep = ssPostInstall then begin
    Browser := 'Chrome';
    if WizardIsTaskSelected('edge') then begin
      if WizardIsTaskSelected('chrome') then Browser := 'Both' else Browser := 'Edge';
    end;
    Params := '-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' + ExpandConstant('{app}\setup\Configure-Installation.ps1') + '" -Root "' + ExpandConstant('{app}') + '" -Browser ' + Browser + ' -Language ' + ActiveLanguage;
    if ExpandConstant('{param:SKIPDEPENDENCIES|0}') = '1' then Params := Params + ' -SkipDependencies';
    WizardForm.StatusLabel.Caption := CustomMessage('Configuring');
    try
      ConfigurationFailed := not ExecAndLogOutput(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'), Params, '', SW_HIDE, ewWaitUntilTerminated, ExitCode, @SetupLog);
      ConfigurationFailed := ConfigurationFailed or (ExitCode <> 0);
    except
      Log(GetExceptionMessage);
      ConfigurationFailed := True;
    end;
    if ConfigurationFailed then begin
      WizardForm.FinishedLabel.Caption := CustomMessage('Incomplete');
      if not WizardSilent then MsgBox(CustomMessage('IncompleteDialog'), mbError, MB_OK);
    end;
  end;
end;

function GetCustomSetupExitCode: Integer;
begin
  if ConfigurationFailed then Result := 1 else Result := 0;
end;

procedure CurPageChanged(CurPageID: Integer);
begin
  if (CurPageID = wpFinished) and ConfigurationFailed then begin
    WizardForm.FinishedHeadingLabel.Caption := CustomMessage('IncompleteHeading');
    WizardForm.FinishedLabel.Caption := CustomMessage('Incomplete');
  end;
end;
