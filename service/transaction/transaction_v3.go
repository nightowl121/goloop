package transaction

import (
	"bytes"
	"encoding/json"
	"log"
	"math/big"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/codec"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/server/jsonrpc"
	"github.com/icon-project/goloop/service/contract"
	"github.com/icon-project/goloop/service/scoreresult"
	"github.com/icon-project/goloop/service/state"
)

const (
	txMaxDataSize                = 512 * 1024 // 512kB
	configCheckDataOnPreValidate = false
)

type transactionV3Data struct {
	Version   common.HexUint16 `json:"version"`
	From      common.Address   `json:"from"`
	To        common.Address   `json:"to"`
	Value     *common.HexInt   `json:"value"`
	StepLimit common.HexInt    `json:"stepLimit"`
	TimeStamp common.HexInt64  `json:"timestamp"`
	NID       *common.HexInt64 `json:"nid,omitempty"`
	Nonce     *common.HexInt   `json:"nonce,omitempty"`
	Signature common.Signature `json:"signature"`
	DataType  *string          `json:"dataType,omitempty"`
	Data      json.RawMessage  `json:"data,omitempty"`
}

func (tx *transactionV3Data) calcHash() ([]byte, error) {
	// sha := sha3.New256()
	sha := bytes.NewBuffer(nil)
	sha.Write([]byte("icx_sendTransaction"))

	// data
	if tx.Data != nil {
		sha.Write([]byte(".data."))
		if len(tx.Data) > 0 {
			var obj interface{}
			if err := json.Unmarshal(tx.Data, &obj); err != nil {
				return nil, err
			}
			if bs, err := serializeValue(obj); err != nil {
				return nil, err
			} else {
				sha.Write(bs)
			}
		}
	}

	// dataType
	if tx.DataType != nil {
		sha.Write([]byte(".dataType."))
		sha.Write([]byte(*tx.DataType))
	}

	// from
	sha.Write([]byte(".from."))
	sha.Write([]byte(tx.From.String()))

	// nid
	if tx.NID != nil {
		sha.Write([]byte(".nid."))
		sha.Write([]byte(tx.NID.String()))
	}

	// nonce
	if tx.Nonce != nil {
		sha.Write([]byte(".nonce."))
		sha.Write([]byte(tx.Nonce.String()))
	}

	// stepLimit
	sha.Write([]byte(".stepLimit."))
	sha.Write([]byte(tx.StepLimit.String()))

	// timestamp
	sha.Write([]byte(".timestamp."))
	sha.Write([]byte(tx.TimeStamp.String()))

	// to
	sha.Write([]byte(".to."))
	sha.Write([]byte(tx.To.String()))

	// value
	if tx.Value != nil {
		sha.Write([]byte(".value."))
		sha.Write([]byte(tx.Value.String()))
	}

	// version
	sha.Write([]byte(".version."))
	sha.Write([]byte(tx.Version.String()))

	return crypto.SHA3Sum256(sha.Bytes()), nil
}

type transactionV3 struct {
	transactionV3Data
	txHash []byte
	bytes  []byte
}

func (tx *transactionV3) Timestamp() int64 {
	return tx.TimeStamp.Value
}

func (tx *transactionV3) verifySignature() error {
	pk, err := tx.Signature.RecoverPublicKey(tx.TxHash())
	if err != nil {
		return InvalidSignatureError.Wrap(err, "FAIL to recover public key")
	}
	addr := common.NewAccountAddressFromPublicKey(pk)
	if addr.Equal(tx.From()) {
		return nil
	}
	return ErrInvalidSignature
}

func (tx *transactionV3) TxHash() []byte {
	if tx.txHash == nil {
		h, err := tx.calcHash()
		if err != nil {
			tx.txHash = []byte{}
		} else {
			tx.txHash = h
		}
	}
	return tx.txHash
}

func (tx *transactionV3) From() module.Address {
	return &tx.transactionV3Data.From
}

func (tx *transactionV3) ID() []byte {
	return tx.TxHash()
}

func (tx *transactionV3) Version() int {
	return module.TransactionVersion3
}

func (tx *transactionV3) Verify(ts int64) error {
	if ConfigOnCheckingTimestamp {
		if ts != 0 {
			tsDiff := tx.TimeStamp.Value - ts
			if tsDiff <= -ConfigTXTimestampBackwardMargin ||
				tsDiff > ConfigTXTimestampForwardLimit {
				return state.TimeOutError.Errorf("Timeout(cur:%d, tx:%d)", ts, tx.TimeStamp.Value)
			}
			if tsDiff > ConfigTXTimestampForwardMargin {
				return state.TimeOutError.Errorf("FutureTxTime(cur:%d, tx:%d)", ts, tx.TimeStamp.Value)
			}
		}
	}
	// value >= 0
	if tx.Value != nil && tx.Value.Sign() < 0 {
		return InvalidTxValue.Errorf("InvalidTxValue(%s)", tx.Value.String())
	}

	// character level size of data element <= 512KB
	n, err := countBytesOfData(tx.Data)
	if err != nil {
		return InvalidTxValue.Wrapf(err, "InvalidData(%x)", tx.Data)
	} else if n > txMaxDataSize {
		return InvalidTxValue.Errorf("InvalidDataSize(%d)", n)
	}

	// Checkups by data types
	if tx.DataType != nil {
		switch *tx.DataType {
		case DataTypeCall:
			// element check
			if tx.Data == nil {
				return InvalidTxValue.Errorf("TxData for call is NIL")
			}
			_, err := ParseCallData(tx.Data)
			return err
		case DataTypeDeploy:
			// element check
			if tx.Data == nil {
				return InvalidTxValue.New("TxData for deploy is NIL")
			}
			type dataDeployJSON struct {
				ContentType string          `json:"contentType"`
				Content     common.HexBytes `json:"content"`
				Params      json.RawMessage `json:"params"`
			}
			var jso dataDeployJSON
			if json.Unmarshal(tx.Data, &jso) != nil || jso.ContentType == "" ||
				jso.Content == nil {
				return InvalidTxValue.Errorf("InvalidDeployData(%s)", string(tx.Data))
			}

			// value == 0
			if tx.Value != nil && tx.Value.Sign() != 0 {
				return InvalidTxValue.Errorf("InvalidTxValue(%s)", tx.Value.String())
			}
		}
	}

	// signature verification
	if err := tx.verifySignature(); err != nil {
		return err
	}

	return nil
}

func (tx *transactionV3) ValidateNetwork(nid int) bool {
	if tx.NID == nil {
		return true
	}
	return int(tx.NID.Value) == nid
}

func (tx *transactionV3) PreValidate(wc state.WorldContext, update bool) error {
	// outdated or invalid timestamp?
	if ConfigOnCheckingTimestamp {
		tsDiff := tx.TimeStamp.Value - wc.BlockTimeStamp()
		if tsDiff <= -ConfigTXTimestampBackwardMargin ||
			tsDiff > ConfigTXTimestampForwardLimit {
			return state.TimeOutError.Errorf("Timeout(block:%d, tx:%d)", wc.BlockTimeStamp(), tx.TimeStamp.Value)
		}
		if tsDiff > ConfigTXTimestampForwardMargin {
			return state.TimeOutError.Errorf("FutureTxTime(block:%d, tx:%d)", wc.BlockTimeStamp(), tx.TimeStamp.Value)
		}
	}

	// stepLimit >= default step + input steps
	cnt, err := measureBytesOfData(wc.Revision(), tx.Data)
	if err != nil {
		return err
	}
	minStep := big.NewInt(wc.StepsFor(state.StepTypeDefault, 1) + wc.StepsFor(state.StepTypeInput, cnt))
	if tx.StepLimit.Cmp(minStep) < 0 {
		return state.NotEnoughStepError.Errorf("NotEnoughStep(txStepLimit:%s, minStep:%s)\n", tx.StepLimit, minStep)
	}

	// balance >= (fee + value)
	stepPrice := wc.StepPrice()

	trans := new(big.Int)
	trans.Set(&tx.StepLimit.Int)
	trans.Mul(trans, stepPrice)
	if tx.Value != nil {
		trans.Add(trans, &tx.Value.Int)
	}

	as1 := wc.GetAccountState(tx.From().ID())
	balance1 := as1.GetBalance()
	if balance1.Cmp(trans) < 0 {
		return scoreresult.Errorf(module.StatusOutOfBalance, "OutOfBalance(balance:%s, value:%s)\n", balance1, trans)
	}

	// for cumulative balance check
	if update {
		as2 := wc.GetAccountState(tx.To.ID())
		balance2 := as2.GetBalance()
		if tx.Value != nil {
			balance2.Add(balance2, &tx.Value.Int)
		}
		balance1.Sub(balance1, trans)
		as1.SetBalance(balance1)
		as2.SetBalance(balance2)
	}

	// checkups by data types
	if configCheckDataOnPreValidate && tx.DataType != nil {
		switch *tx.DataType {
		case DataTypeCall:
			// check if contract is active and not blacklisted
			as := wc.GetAccountState(tx.To.ID())
			if !as.IsContract() {
				return contract.InvalidContractError.New("NotAContractAccount")
			}
			if as.ActiveContract() == nil {
				return contract.InvalidContractError.Errorf(
					"NotActiveContract(blocked(%t), disabled(%t))", as.IsBlocked(), as.IsDisabled())
			}

			// check method and parameters
			if info := as.APIInfo(); info == nil {
				return state.ErrNoActiveContract
			} else {
				jso, _ := ParseCallData(tx.Data) // Already checked at Verify(). It can't be nil.
				if _, err = info.ConvertParamsToTypedObj(jso.Method, jso.Params); err != nil {
					return err
				}
			}
		case DataTypeDeploy:
			// update case: check if contract is active and from is its owner
			if !bytes.Equal(tx.To.ID(), state.SystemID) { // update
				as := wc.GetAccountState(tx.To.ID())
				if !as.IsContract() {
					return contract.InvalidContractError.New("NotAContractAccount")
				}
				if as.ActiveContract() == nil {
					return contract.InvalidContractError.Errorf(
						"NotActiveContract(blocked(%t), disabled(%t))", as.IsBlocked(), as.IsDisabled())
				}
				if !as.IsContractOwner(tx.From()) {
					return scoreresult.New(module.StatusAccessDenied, "NotContractOwner")
				}
			}
		}
	}
	return nil
}

func (tx *transactionV3) GetHandler(cm contract.ContractManager) (TransactionHandler, error) {
	var value *big.Int
	if tx.Value != nil {
		value = &tx.Value.Int
	} else {
		value = big.NewInt(0)
	}
	return NewTransactionHandler(cm,
		tx.From(),
		&tx.To,
		value,
		&tx.StepLimit.Int,
		tx.DataType,
		tx.Data)
}

func (tx *transactionV3) Group() module.TransactionGroup {
	return module.TransactionGroupNormal
}

func (tx *transactionV3) Bytes() []byte {
	if tx.bytes == nil {
		if bs, err := codec.MarshalToBytes(&tx.transactionV3Data); err != nil {
			log.Panicf("Fail to marshal transaction=%+v err=%+v", tx, err)
			return nil
		} else {
			tx.bytes = bs
		}
	}
	return tx.bytes
}

func (tx *transactionV3) SetBytes(bs []byte) error {
	_, err := codec.UnmarshalFromBytes(bs, &tx.transactionV3Data)
	if err != nil {
		return ErrInvalidFormat
	}
	if tx.transactionV3Data.Version.Value != module.TransactionVersion3 {
		return InvalidVersion.Errorf("NotTxVersion3(%d)", tx.transactionV3Data.Version.Value)
	}
	nbs := make([]byte, len(bs))
	copy(nbs, bs)
	tx.bytes = nbs
	return nil
}

func (tx *transactionV3) Hash() []byte {
	return crypto.SHA3Sum256(tx.Bytes())
}

func (tx *transactionV3) Nonce() *big.Int {
	if nonce := tx.transactionV3Data.Nonce; nonce != nil {
		return &nonce.Int
	}
	return nil
}

func (tx *transactionV3) ToJSON(version int) (interface{}, error) {
	if version == jsonrpc.APIVersion3 {
		jso := map[string]interface{}{
			"version":   &tx.transactionV3Data.Version,
			"from":      &tx.transactionV3Data.From,
			"to":        &tx.transactionV3Data.To,
			"stepLimit": &tx.transactionV3Data.StepLimit,
			"timestamp": &tx.transactionV3Data.TimeStamp,
			"signature": &tx.transactionV3Data.Signature,
		}
		if tx.transactionV3Data.Value != nil {
			jso["value"] = tx.transactionV3Data.Value
		}
		if tx.transactionV3Data.NID != nil {
			jso["nid"] = tx.transactionV3Data.NID
		}
		if tx.transactionV3Data.Nonce != nil {
			jso["nonce"] = tx.transactionV3Data.Nonce
		}
		if tx.transactionV3Data.DataType != nil {
			jso["dataType"] = *tx.transactionV3Data.DataType
		}
		if tx.transactionV3Data.Data != nil {
			jso["data"] = json.RawMessage(tx.transactionV3Data.Data)
		}
		jso["txHash"] = common.HexBytes(tx.ID())

		return jso, nil
	} else {
		return nil, InvalidVersion.Errorf("Version(%d)", version)
	}
}

func (tx *transactionV3) MarshalJSON() ([]byte, error) {
	if obj, err := tx.ToJSON(jsonrpc.APIVersionLast); err != nil {
		return nil, scoreresult.WithStatus(err, module.StatusIllegalFormat)
	} else {
		return json.Marshal(obj)
	}
}

func newTransactionV3FromJSONObject(jso *transactionV3JSON) (Transaction, error) {
	tx := new(transactionV3)
	tx.transactionV3Data = jso.transactionV3Data
	return tx, nil
}

func newTransactionV3FromBytes(bs []byte) (Transaction, error) {
	tx := new(transactionV3)
	if err := tx.SetBytes(bs); err != nil {
		return nil, scoreresult.WithStatus(err, module.StatusIllegalFormat)
	} else {
		return tx, nil
	}
}

func measureBytesOfData(rev int, data []byte) (int, error) {
	if data == nil {
		return 0, nil
	}

	if rev >= module.Revision3 {
		return countBytesOfData(data)
	} else {
		var idata interface{}
		if err := json.Unmarshal(data, &idata); err != nil {
			return 0, scoreresult.WithStatus(err, module.StatusIllegalFormat)
		} else {
			return countBytesOfDataValue(idata), nil
		}
	}
}

func countBytesOfData(data []byte) (int, error) {
	if data == nil {
		return 0, nil
	}
	b := bytes.NewBuffer(nil)
	if err := json.Compact(b, data); err != nil {
		return 0, scoreresult.ErrInvalidParameter
	}
	return b.Len(), nil
}

func countBytesOfDataValue(v interface{}) int {
	switch o := v.(type) {
	case string:
		if len(o) > 2 && o[:2] == "0x" {
			o = o[2:]
		}
		bs := []byte(o)
		for _, b := range bs {
			if (b < '0' || b > '9') && (b < 'a' || b > 'f') {
				return len(bs)
			}
		}
		return (len(bs) + 1) / 2
	case []interface{}:
		var count int
		for _, i := range o {
			count += countBytesOfDataValue(i)
		}
		return count
	case map[string]interface{}:
		var count int
		for _, i := range o {
			count += countBytesOfDataValue(i)
		}
		return count
	case bool:
		return 1
	case float64:
		return len(common.Int64ToBytes(int64(o)))
	default:
		return 0
	}
}
