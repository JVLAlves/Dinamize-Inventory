package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/JVLAlves/Dinamize-Inventory/DataMission/Linux"
	"github.com/JVLAlves/Dinamize-Inventory/DataMission/MacOS"
	"github.com/JVLAlves/Dinamize-Inventory/DataMission/Windows"
	functions "github.com/JVLAlves/Dinamize-Inventory/Utilities/Functions"
	snipe "github.com/JVLAlves/Dinamize-Inventory/Utilities/SnipeMethods"
	globals "github.com/JVLAlves/Dinamize-Inventory/cmd"
	"github.com/JVLAlves/Dinamize-Inventory/regexs"
)

//Função de execução do programa em MacOS
func forMacOs(f *os.File) {

	//Criando Arquivos via Goroutines
	wg := &sync.WaitGroup{}
	wg.Add(5)
	go MacOS.Create(wg, "uname", "-n")
	go MacOS.Create(wg, "sysctl", "-a |grep machdep.cpu.brand_string |awk '{print $2,$3,$4}'")
	go MacOS.Create(wg, "hostinfo", "|grep memory |awk '{print $4,$5}'")
	go MacOS.Create(wg, "diskutil", "list |grep disk0s2 | awk '{print $5,$6}'")
	go MacOS.Create(wg, "sw_vers", "-productVersion")
	wg.Wait()

	//Realiza o processo de coleta de dados do Sistema MacOS e retorna as informações em um array Infos
	MacOS.Running()

	//Variavel de Contrato
	mac := snipe.NewActive()

	//Populando Struct //tALVEZ PODE SER FEITA MELHOR, NÃO SEI.
	mac.SnipeitCPU11 = MacOS.Infos[1]
	mac.SnipeitHostname10 = MacOS.Infos[0]
	mac.Name = MacOS.Infos[0]

	//Passando Regex antes de popular informação de Memória

	MemoryRegex := regexs.RegexHDandMemory.FindStringSubmatch(MacOS.Infos[2])
	//Convertendo response de string para float
	MemoryFloat, _ := strconv.ParseFloat(MemoryRegex[1], 64)
	//Arredondando valor númerico da variável
	MemoryRounded := math.Round(MemoryFloat)
	//Populando campo de memória com o valor tratado
	mac.SnipeitMema3Ria7 = strconv.Itoa(int(MemoryRounded)) + "GB"

	//Passando Regex antes de popular informação de HD
	HDRegex := regexs.RegexHDandMemory.FindStringSubmatch(MacOS.Infos[3])
	//Convertendo response de string para float
	HDFloat, _ := strconv.ParseFloat(HDRegex[1], 64)
	//Arredondando valor númerico da variável
	HDRounded := math.Round(HDFloat)
	//Populando campo de HD com o valor tratado
	mac.SnipeitHd9 = strconv.Itoa(int(HDRounded)) + "GB"

	//Passando Regex antes de popular informação de Asset Tag
	mac.AssetTag = regexs.RegexAssettagDigit.FindString(MacOS.Infos[0])
	//Caso não haja digitos no campo HOSTNAME (Fonte do Asset Tag), o retorno do sistema é um Asset Tag Default (NO ASSET TAG)
	if mac.AssetTag == "" {
		mac.AssetTag = "No Asset Tag"
		fmt.Fprintf(f, "Nenhum Asset Tag foi definido, pois nenhuma sequência numérica foi encontrada no HOSTNAME: %v", MacOS.Infos[0])

	}

	//Passando Regex antes de popular informação de Sistema Operacional
	SORegex := regexs.RegexMacOS.FindStringSubmatch(MacOS.Infos[4])
	//Convertendo response de string para float
	SOFloat, err := strconv.ParseFloat(SORegex[1], 64)
	//Tratando erro
	if err != nil {
		log.Fatalf("Erro na conversão do S.O. para float")
	}

	//Verificação de Versão Menores (11.5.1) e substituição por Versões Maiores (11.4)

	var SOString string
	if SOFloat >= 11.4 && SOFloat < 12.0 {
		SOString = "11.4"
	} else {
		SOString = SORegex[1]
	}

	//Alternando Versão Númerica (RETIRADA DO SISTEMA) para Versão Nominal (DEFINIDA PELA APPLE INC.)
	//ISTO PODE SER UM MAP[STRING]STRING. PARA DESCOBRIR A VERSÃO, PASSA-SE UM FOR SOBRE O MAP.
	for in, v := range globals.MacOSVersions {

		if SOString == in {
			mac.SnipeitSo8 = v
			break
		}

	}

	//Entrada Default
	mac.ModelID = globals.ID_MODELO
	mac.StatusID = globals.ID_STATUS
	mac.SnipeitModel12 = globals.MODELO_ATIVO

	//Resumo gráfico das informações coletadas.
	fmt.Printf("NOME DO DISPOSITIVO: %v\n", mac.Name)
	fmt.Printf("ASSET TAG: %v\n", mac.AssetTag)
	fmt.Printf("TIPO DE ATIVO: %v\n", mac.ModelID)
	fmt.Printf("MODELO DO ATIVO: %v\n", mac.SnipeitModel12)
	fmt.Printf("STATUS: %v\n\n", mac.StatusID)
	fmt.Printf("DESCRIÇÃO DO ATIVO\n")
	fmt.Printf("HOSTNAME: %v\n", mac.SnipeitHostname10)
	fmt.Printf("S.O.: %v\n", mac.SnipeitSo8)
	fmt.Printf("CPU: %v\n", mac.SnipeitCPU11)
	fmt.Printf("MEMORIA RAM: %v\n", mac.SnipeitMema3Ria7)
	fmt.Printf("DISCO: %v\n\n", mac.SnipeitHd9)

	//Verificando a existência de um ativo semelhante no inventário Snipe it
	if snipe.Verifybytag(mac.AssetTag, globals.IP_SNIPEIT) {
		fmt.Fprintln(f, "Os dados do Ativo Criado não constam no sistema.")
		fmt.Println("Enviando Ativo para o Snipeit ")

		//Caso o Ativo não exista no sistema, as informações são enviadas para tal.
		snipe.PostSnipe(mac, globals.IP_SNIPEIT, f)
	} else {
		//caso já exista, o programa procura por disparidades.
		//log.Println("Um Ativo semelhante foi encontrado no sistema.")
		fmt.Print("Asset Tag idêntico encontrado. Iniciando análise de disparidades")
		PatchRequestUrl, IsNeeded := snipe.Getbytag(globals.IP_SNIPEIT, mac.AssetTag, mac, f)
		if IsNeeded {
			//Caso haja disparidades, o processo de PATCH é iniciado.
			fmt.Println("\nPATCH necessário.")
			fmt.Println("\nExecutando PATCH RESQUEST.")

			id := snipe.Getidbytag(mac.AssetTag, globals.IP_SNIPEIT)
			snipe.Patchbyid(id, globals.IP_SNIPEIT, PatchRequestUrl)

		} else {
			//Caso não haja disparidades... Nada acontece.
			_, _ = fmt.Fprintf(f, "")
			fmt.Fprintln(f, "\nSem alterações")
		}
	}
}

//Função de execução do programa em Windows
func forWindows(f *os.File) {

	//Realiza o processo de coleta de dados do Sistema Windows e retorna as informações em um array Infos
	Windows.MainProgram()

	//Variavel de Contrato
	win := snipe.NewActive()

	//Populando Struct
	win.SnipeitCPU11 = Windows.Infos[3]
	win.SnipeitMema3Ria7 = Windows.Infos[2]
	win.SnipeitSo8 = Windows.Infos[1]
	win.SnipeitHostname10 = Windows.Infos[0]
	win.Name = Windows.Infos[0]

	//Passando Regex antes de popular informação de Memória (COLETA: Primeiros três digitos com espaço em branco)
	MemoryRegex := regexs.RegexHDandMemory.FindStringSubmatch(Windows.Infos[2])
	//Convertendo response de string para float
	MemoryFloat, _ := strconv.ParseFloat(MemoryRegex[1], 64)
	//Arredondando valor númerico da variável
	MemoryRounded := math.Round(MemoryFloat)
	//Populando campo de memória com o valor tratado
	win.SnipeitMema3Ria7 = strconv.Itoa(int(MemoryRounded)) + "GB"

	//Passando Regex antes de popular informação de HD
	HDRegex := regexs.RegexHDandMemory.FindStringSubmatch(Windows.Infos[4])
	//Convertendo response de string para float
	HDFloat, _ := strconv.ParseFloat(HDRegex[1], 64)
	//Arredondando valor númerico da variável
	HDRounded := math.Round(HDFloat)
	//Populando campo de HD com o valor tratado
	win.SnipeitHd9 = strconv.Itoa(int(HDRounded)) + "GB"

	//Passando Regex antes de popular informação de Asset Tag
	win.AssetTag = regexs.RegexAssettagDigit.FindString(Windows.Infos[0])
	//Caso não haja digitos no campo HOSTNAME (Fonte do Asset Tag), o retorno do sistema é um Asset Tag Default (NO ASSET TAG)
	if win.AssetTag == "" {
		win.AssetTag = "No Asset Tag"
		log.Printf("Nenhum Asset Tag foi defino, pois nenhuma sequência numérica foi encontrada no HOSTNAME: %v", Windows.Infos[0])

	}

	//Entrada Default
	win.ModelID = globals.ID_MODELO
	win.StatusID = globals.ID_STATUS
	win.SnipeitModel12 = globals.MODELO_ATIVO

	//Resumo gráfico das informações coletadas.
	fmt.Printf("NOME DO DISPOSITIVO: %v\n", win.Name)
	fmt.Printf("ASSET TAG: %v\n", win.AssetTag)
	fmt.Printf("TIPO DE ATIVO: %v\n", win.ModelID)
	fmt.Printf("MODELO DO ATIVO: %v\n", win.SnipeitModel12)
	fmt.Printf("STATUS: %v\n\n", win.StatusID)
	fmt.Printf("DESCRIÇÃO DO ATIVO\n")
	fmt.Printf("HOSTNAME: %v\n", win.SnipeitHostname10)
	fmt.Printf("S.O.: %v\n", win.SnipeitSo8)
	fmt.Printf("CPU: %v\n", win.SnipeitCPU11)
	fmt.Printf("MEMORIA RAM: %v\n", win.SnipeitMema3Ria7)
	fmt.Printf("DISCO: %v\n\n", win.SnipeitHd9)

	//Verificando a existência de um ativo semelhante no inventário Snipe it
	if snipe.Verifybytag(win.AssetTag, globals.IP_SNIPEIT) {
		fmt.Fprintln(f, "Os dados do Ativo Criado não constam no sistema.")
		fmt.Println("Enviando Ativo para o Snipeit ")

		//Caso o Ativo não exista no sistema, as informações são enviadas para tal.
		snipe.PostSnipe(win, globals.IP_SNIPEIT, f)

		log.Println("Ativo Criado enviado para o sistema.")

	} else {
		//caso já exista, o programa procura por disparidades.
		//log.Println("Um Ativo semelhante foi encontrado no sistema.")
		fmt.Print("Asset Tag idêntico encontrado. Iniciando análise de disparidades")
		PatchRequestUrl, IsNeeded := snipe.Getbytag(globals.IP_SNIPEIT, win.AssetTag, win, f)
		if IsNeeded {
			//Caso haja disparidades, o processo de PATCH é iniciado.
			fmt.Println("\nPATCH necessário.")
			fmt.Println("\nExecutando PATCH RESQUEST.")

			id := snipe.Getidbytag(win.AssetTag, globals.IP_SNIPEIT)
			snipe.Patchbyid(id, globals.IP_SNIPEIT, PatchRequestUrl)

		} else {
			//Caso não haja disparidades... Nada acontece.
			fmt.Fprintln(f, "\nSem alterações")
		}
	}

}

//Função de execução do programa em Linux
func forLinux(f *os.File) {

	//Realiza o processo de coleta de dados do Sistema Linux e retorna as informações em um array Infos
	Linux.MainProgram()

	//Variavel de Contrato
	lin := snipe.NewActive()

	//Populando Struct
	lin.SnipeitCPU11 = Linux.Infos[0]
	lin.SnipeitSo8 = Linux.Infos[2]
	lin.SnipeitHostname10 = Linux.Infos[3]
	lin.Name = Linux.Infos[3]

	//Passando Regex antes de popular informação de HD (COLETA: Número com vírgula)
	HDRegex := regexs.RegexHDandMemory.FindStringSubmatch(Linux.Infos[4])
	//Separação do result
	HDSplitted := strings.Split(HDRegex[1], ",")
	//Integração do result utilizando ponto (Padrão para conversão)
	HDJoined := strings.Join(HDSplitted, ".")
	//Convertendo response de string para float
	HDFloat, _ := strconv.ParseFloat(HDJoined, 64)
	//Arredondando valor númerico da variável
	HDRounded := math.Round(HDFloat)
	//Populando campo de HD com o valor tratado
	lin.SnipeitHd9 = strconv.Itoa(int(HDRounded)) + "GB"

	//Passando Regex antes de popular informação de Memória
	MemoryRegex := regexs.RegexHDandMemory.FindStringSubmatch(Linux.Infos[1])
	//Convertendo response de string para float
	MemoryFloat, _ := strconv.ParseFloat(MemoryRegex[1], 64)
	//Arredondando valor númerico da variável
	MemoryRounded := math.Round(MemoryFloat)
	//Populando campo de memória com o valor tratado
	lin.SnipeitMema3Ria7 = strconv.Itoa(int(MemoryRounded)) + "GB"

	//Passando Regex antes de popular informação de Asset Tag
	lin.AssetTag = functions.RegexThis(`\d`, Linux.Infos[3])
	//Caso não haja digitos no campo HOSTNAME (Fonte do Asset Tag), o retorno do sistema é um Asset Tag Default (NO ASSET TAG)
	if lin.AssetTag == "" {
		lin.AssetTag = "No Asset Tag"
		log.Printf("Nenhum Asset Tag foi defino, pois nenhuma sequência numérica foi encontrada no HOSTNAME: %v", Linux.Infos[0])

	}

	//Entrada Default
	lin.ModelID = globals.ID_MODELO
	lin.StatusID = globals.ID_STATUS
	lin.SnipeitModel12 = globals.MODELO_ATIVO

	//Resumo gráfico das informações coletadas.
	fmt.Printf("NOME DO DISPOSITIVO: %v\n", lin.Name)
	fmt.Printf("ASSET TAG: %v\n", lin.AssetTag)
	fmt.Printf("TIPO DE ATIVO: %v\n", lin.ModelID)
	fmt.Printf("MODELO DO ATIVO: %v\n", lin.SnipeitModel12)
	fmt.Printf("STATUS: %v\n\n", lin.StatusID)
	fmt.Printf("DESCRIÇÃO DO ATIVO\n")
	fmt.Printf("HOSTNAME: %v\n", lin.SnipeitHostname10)
	fmt.Printf("S.O.: %v\n", lin.SnipeitSo8)
	fmt.Printf("CPU: %v\n", lin.SnipeitCPU11)
	fmt.Printf("MEMORIA RAM: %v\n", lin.SnipeitMema3Ria7)
	fmt.Printf("DISCO: %v\n\n", lin.SnipeitHd9)

	//Verificando a existência de um ativo semelhante no inventário Snipe it
	if snipe.Verifybytag(lin.AssetTag, globals.IP_SNIPEIT) {
		fmt.Fprintln(f, "Os dados do Ativo Criado não constam no sistema.")
		fmt.Println("Enviando Ativo para o Snipeit ")

		//Caso o Ativo não exista no sistema, as informações são enviadas para tal.
		snipe.PostSnipe(lin, globals.IP_SNIPEIT, f)

	} else {
		//caso já exista, o programa procura por disparidades.
		//log.Println("Um Ativo semelhante foi encontrado no sistema.")
		fmt.Print("Asset Tag idêntico encontrado. Iniciando análise de disparidades")
		PatchRequestUrl, IsNeeded := snipe.Getbytag(globals.IP_SNIPEIT, lin.AssetTag, lin, f)
		if IsNeeded {
			//Caso haja disparidades, o processo de PATCH é iniciado.
			fmt.Println("\nPATCH necessário.")
			fmt.Println("\nExecutando PATCH RESQUEST.")

			id := snipe.Getidbytag(lin.AssetTag, globals.IP_SNIPEIT)
			snipe.Patchbyid(id, globals.IP_SNIPEIT, PatchRequestUrl)

		} else {
			//Caso não haja disparidades... Nada acontece.
			fmt.Fprintln(f, "\nSem alterações")
		}
	}

}

//função Principal do programa
func main() {

	//Cria tanto a pasta para logs quanto o arquivo inicial de logs
	f := functions.ActiveLogs()

	//Log de inicialização
	log.Printf("Inicio de execução.")

	//Identificando sistema operacional
	switch runtime.GOOS {
	case "darwin":
		forMacOs(f)
	case "linux":
		forLinux(f)

	case "windows":
		forWindows(f)
	default:
		log.Fatalf("Erro em econtrar o Sistema Operacional")
	}

	//mensagem de encerramento
	fmt.Println("\n\nObrigado pela paciência! (FIM)")
	log.Printf("Fim de execução.")
}