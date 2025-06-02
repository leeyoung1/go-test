package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuperchain/xuper-sdk-go/v2/account"
	"github.com/xuperchain/xuper-sdk-go/v2/xuper"
)

const (
	nodeAddr        = "127.0.0.1:37101" // 超级链节点地址
	chainName       = "xuper"           // 链名
	keysDir         = "./keys"          // 存储账户密钥的目录
	contractCodeDir = "./contract"      // 存储编译好的合约代码的目录
)

var sdkClient *xuper.XClient
var currentAccount *account.Account

func main() {
	var err error
	sdkClient, err = xuper.New(nodeAddr)
	if err != nil {
		fmt.Printf("连接超级链节点失败: %v\n", err)
		return
	}
	defer sdkClient.Close()

	fmt.Println("成功连接到超级链节点:", nodeAddr)

	// 确保keys目录存在
	if _, err := os.Stat(keysDir); os.IsNotExist(err) {
		os.MkdirAll(keysDir, 0755)
	}
	// 确保合约代码目录存在
	if _, err := os.Stat(contractCodeDir); os.IsNotExist(err) {
		os.MkdirAll(contractCodeDir, 0755)
		fmt.Printf("请将编译好的智能合约文件 (例如 digital_copyright_service) 放入 %s 目录\n", contractCodeDir)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		printMenu()
		fmt.Print("请输入选项: ")
		input, _ := reader.ReadString('\n')
		choice, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil {
			fmt.Println("无效输入，请输入数字。")
			continue
		}

		handleChoice(choice, reader)
	}
}

func printMenu() {
	fmt.Println("\n--- 超级链客户端菜单 ---")
	if currentAccount != nil {
		fmt.Printf("--- 当前账户: %s ---\n", currentAccount.Address)
	} else {
		fmt.Println("--- 当前未加载账户 ---")
	}
	fmt.Println("1. 创建新账户 (并保存到文件)")
	fmt.Println("2. 从文件加载账户 (设为当前账户)")
	fmt.Println("3. 查询当前账户余额")
	fmt.Println("4. 部署数字版权合约")
	fmt.Println("--- 合约调用 (需先部署合约并加载账户) ---")
	fmt.Println("5. 注册数字版权")
	fmt.Println("6. 查询数字版权")
	fmt.Println("7. 转移数字版权")
	fmt.Println("8. 更新版权描述")
	fmt.Println("9. 删除数字版权 (逻辑删除)")
	fmt.Println("--- 链上查询 ---")
	fmt.Println("10. 查询指定区块信息 (按高度)")
	fmt.Println("11. 查询指定交易信息 (按TxID)")
	fmt.Println("12. 查询链状态")
	fmt.Println("0. 退出")
	fmt.Println("--------------------")
}

func handleChoice(choice int, reader *bufio.Reader) {
	switch choice {
	case 1:
		createAndSaveAccount(reader)
	case 2:
		loadAccountFromFile(reader)
	case 3:
		queryCurrentAccountBalance()
	case 4:
		deployCopyrightContract(reader)
	case 5:
		invokeRegisterCopyright(reader)
	case 6:
		invokeQueryCopyright(reader)
	case 7:
		invokeTransferCopyright(reader)
	case 8:
		invokeUpdateCopyrightDescription(reader)
	case 9:
		invokeDeleteCopyright(reader)
	case 10:
		queryBlockByHeight(reader)
	case 11:
		queryTxByID(reader)
	case 12:
		queryChainStatus()
	case 0:
		fmt.Println("正在退出...")
		os.Exit(0)
	default:
		fmt.Println("无效选项，请重试。")
	}
}

func getStringInput(prompt string, reader *bufio.Reader) string {
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func createAndSaveAccount(reader *bufio.Reader) {
	strengthStr := getStringInput("输入助记词强度 (1:弱, 2:中, 3:强, 默认2): ", reader)
	strength, err := strconv.Atoi(strengthStr)
	if err != nil || strength < 1 || strength > 3 {
		strength = 2 // 默认为中
	}

	langStr := getStringInput("输入助记词语言 (1:中文, 2:英文, 默认2): ", reader)
	language, err := strconv.Atoi(langStr)
	if err != nil || (language != 1 && language != 2) {
		language = 2 // 默认为英文
	}

	password := getStringInput("输入加密账户文件的密码 (可留空): ", reader)

	acc, err := account.CreateAndSaveAccountToFile(keysDir, password, uint8(strength), language)
	if err != nil {
		fmt.Printf("创建并保存账户失败: %v\n", err)
		return
	}
	fmt.Println("账户创建成功!")
	fmt.Printf("  地址: %s\n", acc.Address)
	fmt.Printf("  助记词: %s\n", acc.Mnemonic)
	fmt.Printf("  私钥 (JSON): %s\n", acc.PrivateKey)
	fmt.Printf("  公钥 (JSON): %s\n", acc.PublicKey)
	fmt.Printf("  账户文件保存在: %s/%s.json\n", keysDir, acc.Address)

	currentAccount = acc // 将新创建的账户设为当前账户
	fmt.Println("新创建的账户已设为当前操作账户。")
	fmt.Println("注意: 新账户余额为0，进行交易前请确保有足够余额。")
}

func loadAccountFromFile(reader *bufio.Reader) {
	address := getStringInput("输入要加载的账户地址: ", reader)
	password := getStringInput("输入账户文件密码 (如果设置过): ", reader)

	filePath := filepath.Join(keysDir, address+".json")
	acc, err := account.GetAccountFromFile(filePath, password)
	if err != nil {
		fmt.Printf("从文件加载账户 %s 失败: %v\n", address, err)
		return
	}
	currentAccount = acc
	fmt.Printf("账户 %s 加载成功，并设为当前操作账户。\n", acc.Address)
}

func queryCurrentAccountBalance() {
	if currentAccount == nil {
		fmt.Println("请先加载或创建账户。")
		return
	}
	balance, err := sdkClient.QueryBalance(currentAccount.Address)
	if err != nil {
		fmt.Printf("查询账户 %s 余额失败: %v\n", currentAccount.Address, err)
		return
	}
	fmt.Printf("账户 %s 的余额: %s\n", currentAccount.Address, balance.String())
}

func deployCopyrightContract(reader *bufio.Reader) {
	if currentAccount == nil {
		fmt.Println("请先加载或创建账户用于部署合约。")
		return
	}

	contractName := getStringInput("输入合约名称 (例如 copyright123): ", reader)
	if contractName == "" {
		fmt.Println("合约名称不能为空。")
		return
	}

	codeFilePath := getStringInput(fmt.Sprintf("输入编译好的合约文件路径 (相对于 %s 目录, 例如 digital_copyright_service): ", contractCodeDir), reader)
	fullCodePath := filepath.Join(contractCodeDir, codeFilePath)
	codeBytes, err := ioutil.ReadFile(fullCodePath)
	if err != nil {
		fmt.Printf("读取合约代码文件 %s 失败: %v\n", fullCodePath, err)
		return
	}

	// DeployNativeGoContract 的 args 参数是 map[string]string
	initArgsMap := map[string]string{
		"creator": currentAccount.Address,
	}

	tx, err := sdkClient.DeployNativeGoContract(currentAccount, contractName, codeBytes, initArgsMap)
	if err != nil {
		fmt.Printf("部署合约 %s 失败: %v\n", contractName, err)
		return
	}
	fmt.Printf("合约 %s 部署交易已发送, TxID: %s\n", contractName, tx.Tx.Txid)
	fmt.Println("请稍后查询交易状态以确认部署成功。")
}

// --- 合约调用函数 ---
// 修正: args 参数类型为 map[string][]byte
func invokeContract(methodName string, argsByteMap map[string][]byte, reader *bufio.Reader, isQuery bool) {
	if currentAccount == nil {
		fmt.Println("请先加载或创建账户。")
		return
	}
	contractName := getStringInput("输入要调用的合约名称: ", reader)
	if contractName == "" {
		fmt.Println("合约名称不能为空。")
		return
	}

	if isQuery {
		// 转换 map[string][]byte 为 map[string]string
		argsStringMap := make(map[string]string)
		for k, v := range argsByteMap {
			argsStringMap[k] = string(v)
		}
		preExeRes, err := sdkClient.QueryNativeContract(currentAccount, contractName, methodName, argsStringMap)
		if err != nil {
			fmt.Printf("查询合约 %s::%s 失败: %v\n", contractName, methodName, err)
			return
		}
		// preExeRes.Response 是 *pb.ContractResponse 类型
		if preExeRes.ContractResponse.Status >= 400 { // 假设状态码 >= 400 表示错误
			fmt.Printf("合约查询执行失败: %s\n", preExeRes.ContractResponse.Message)
			if len(preExeRes.ContractResponse.Body) > 0 {
				fmt.Printf("合约返回Body: %s\n", string(preExeRes.ContractResponse.Body))
			}
			return
		}
		fmt.Printf("合约查询 %s::%s 成功:\n", contractName, methodName)
		fmt.Printf("  Gas消耗: %d\n", preExeRes.GasUsed) // 直接访问 GasUsed
		fmt.Printf("  返回结果: %s\n", string(preExeRes.ContractResponse.Body))

		// 尝试格式化JSON输出
		var prettyJSON map[string]interface{}
		if json.Unmarshal(preExeRes.ContractResponse.Body, &prettyJSON) == nil {
			formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
			fmt.Printf("  格式化结果:\n%s\n", string(formatted))
		}

	} else { // Invoke (写操作)
		// 转换 map[string][]byte 为 map[string]string
		argsStringMap := make(map[string]string)
		for k, v := range argsByteMap {
			argsStringMap[k] = string(v)
		}
		tx, err := sdkClient.InvokeNativeContract(currentAccount, contractName, methodName, argsStringMap)
		if err != nil {
			fmt.Printf("调用合约 %s::%s 失败: %v\n", contractName, methodName, err)
			return
		}
		fmt.Printf("合约调用 %s::%s 交易已发送, TxID: %s\n", contractName, methodName, tx.Tx.Txid)
		fmt.Println("请稍后查询交易状态以确认。")
	}
}

func invokeRegisterCopyright(reader *bufio.Reader) {
	fmt.Println("--- 注册数字版权 ---")
	// 构造 map[string][]byte 类型的参数
	args := map[string][]byte{
		"copyright_id":  []byte(getStringInput("输入版权ID: ", reader)),
		"work_name":     []byte(getStringInput("输入作品名称: ", reader)),
		"author":        []byte(getStringInput("输入作者: ", reader)),
		"owner_address": []byte(getStringInput("输入初始持有人地址: ", reader)),
		"work_type":     []byte(getStringInput("输入作品类型: ", reader)),
		"description":   []byte(getStringInput("输入作品描述: ", reader)),
	}
	invokeContract("RegisterCopyright", args, reader, false)
}

func invokeQueryCopyright(reader *bufio.Reader) {
	fmt.Println("--- 查询数字版权 ---")
	args := map[string][]byte{
		"copyright_id": []byte(getStringInput("输入要查询的版权ID: ", reader)),
	}
	invokeContract("QueryCopyright", args, reader, true)
}

func invokeTransferCopyright(reader *bufio.Reader) {
	fmt.Println("--- 转移数字版权 ---")
	args := map[string][]byte{
		"copyright_id":      []byte(getStringInput("输入要转移的版权ID: ", reader)),
		"new_owner_address": []byte(getStringInput("输入新持有人地址: ", reader)),
	}
	invokeContract("TransferCopyright", args, reader, false)
}

func invokeUpdateCopyrightDescription(reader *bufio.Reader) {
	fmt.Println("--- 更新版权描述 ---")
	args := map[string][]byte{
		"copyright_id":    []byte(getStringInput("输入要更新的版权ID: ", reader)),
		"new_description": []byte(getStringInput("输入新的作品描述: ", reader)),
	}
	invokeContract("UpdateCopyrightDescription", args, reader, false)
}

func invokeDeleteCopyright(reader *bufio.Reader) {
	fmt.Println("--- 删除数字版权 (逻辑删除) ---")
	args := map[string][]byte{
		"copyright_id": []byte(getStringInput("输入要删除的版权ID: ", reader)),
	}
	invokeContract("DeleteCopyright", args, reader, false)
}

// --- 链上查询函数 ---
func queryBlockByHeight(reader *bufio.Reader) {
	heightStr := getStringInput("输入要查询的区块高度: ", reader)
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		fmt.Println("无效的区块高度。")
		return
	}

	block, err := sdkClient.QueryBlockByHeight(height)
	if err != nil {
		fmt.Printf("查询区块 %d 失败: %v\n", height, err)
		return
	}
	fmt.Printf("--- 区块信息 (高度: %d) ---\n", height)
	fmt.Printf("  区块ID: %s\n", block.Blockid)
	// GetPreHash 返回 []byte, 使用 %x 格式化为十六进制字符串
	fmt.Printf("  前序区块ID: %x\n", block.Block.GetPreHash())
	fmt.Printf("  交易数量: %d\n", len(block.Block.GetTransactions()))
	fmt.Printf("  时间戳: %d\n", block.Block.GetTimestamp())
	// 可以打印更多信息
	// blockJSON, _ := json.MarshalIndent(block, "", "  ")
	// fmt.Println(string(blockJSON))
}

func queryTxByID(reader *bufio.Reader) {
	txID := getStringInput("输入要查询的交易ID: ", reader)
	// QueryTxByID 在 SDK 中接收的 txID 是十六进制字符串
	tx, err := sdkClient.QueryTxByID(txID)
	if err != nil {
		fmt.Printf("查询交易 %s 失败: %v\n", txID, err)
		return
	}
	fmt.Printf("--- 交易信息 (TxID: %s) ---\n", txID)
	// 打印交易的简要信息
	// GetBlockid 返回 []byte
	fmt.Printf("  区块ID: %x\n", tx.GetBlockid())
	fmt.Printf("  发起者: %s\n", tx.GetInitiator()) // GetInitiator 返回 string
	// 尝试打印合约调用信息
	if len(tx.GetContractRequests()) > 0 {
		for i, req := range tx.GetContractRequests() {
			fmt.Printf("  合约请求[%d]:\n", i)
			fmt.Printf("    合约名: %s\n", req.GetContractName())
			fmt.Printf("    方法名: %s\n", req.GetMethodName())
			// req.GetArgs() 返回 map[string][]byte
			// 如果需要打印，可以迭代并打印每个参数
			// argBytes, _ := json.Marshal(req.GetArgs())
			// fmt.Printf("    参数 (JSON): %s\n", string(argBytes))
		}
	}
	// txJSON, _ := json.MarshalIndent(tx, "", "  ")
	// fmt.Println(string(txJSON))
}

func queryChainStatus() {
	status, err := sdkClient.QueryBlockChainStatus(chainName)
	if err != nil {
		fmt.Printf("查询链 %s 状态失败: %v\n", chainName, err)
		return
	}
	fmt.Printf("--- 链状态 (%s) ---\n", chainName)
	fmt.Printf("  最新区块高度: %d\n", status.GetMeta().GetTrunkHeight())
	// GetTipBlockid 返回 []byte
	fmt.Printf("  最新区块ID: %x\n", status.GetMeta().GetTipBlockid())
	// statusJSON, _ := json.MarshalIndent(status, "", "  ")
	// fmt.Println(string(statusJSON))
}
