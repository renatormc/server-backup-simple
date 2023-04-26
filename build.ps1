go1.20 build -o ".\dist\server-backup-simple.exe"
if($? -And ($args.Count -gt 0)){
    .\dist\server-backup-simple.exe $args
}