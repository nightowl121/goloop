package testcases;

import score.*;
import score.annotation.EventLog;
import score.annotation.External;
import score.annotation.Payable;

import java.math.BigInteger;
import java.util.List;
import java.util.Map;

public class IISSTest {
    private String name;
   // private Map<String, Object> map = new HashMap<String, Object>();
    private static final Address CHAIN_SCORE = Address.fromString("cx0000000000000000000000000000000000000000");
    @EventLog
    public void EmitEvent(String data) {}

    public IISSTest(String name) {
        this.name = name;
    }

/*    @External
    public  Object getStake() {
        Address system_address = score.Address.fromString("cx0000000000000000000000000000000000000000");
        return Context.call(system_address, "getStake");
    }
*/
    @Payable
    @External
    public  void setStake(BigInteger value) {
        Object obj = Context.call(CHAIN_SCORE, "setStake", value);
    }

    @External
    public  void setDelegation(Address address, BigInteger value) {
        Delegation [] delegations = new Delegation[1];
        delegations[0] = new Delegation();
        delegations[0].setAddress(address);
        delegations[0].setValue(value);
        Object obj = Context.call(CHAIN_SCORE, "setDelegation", delegations);
    }

    @External(readonly = true)
    public  Map getStakeByScore(Address address) {
        Map<String, Object> map = (Map<String, Object>)Context.call(CHAIN_SCORE, "getStake", address);
        return map;
    }

    @External
    public  void getBalance() {
        BigInteger bal = Context.getBalance(Context.getCaller());
        Context.println("balance : " + bal.toString());
    }

    @External(readonly = true)
    public  Map getPRepByScore(Address address) {
        Map<String, Object> map  = (Map<String, Object>)Context.call(CHAIN_SCORE, "getPRep", address);
        return map;
    }

    @Payable
    @External
    public  void registerPrepByScore(String name, String email, String country, String city, String website, String details, String p2pEndpoint, Address nodeAddress) {
        BigInteger fee = new BigInteger("2000000000000000000000");
        Object obj = Context.call(fee, CHAIN_SCORE, "registerPRep", name, email, country, city, website, details, p2pEndpoint, nodeAddress);
        EmitEvent(nodeAddress.toString());
    }

    @External
    public  void unregisterPRep() {
        Object obj = Context.call(CHAIN_SCORE, "unregisterPRep");
    }

    class Delegation {
        private Address address;
        private BigInteger value;

        public Delegation() {}

        public Address getAddress() {
            return address;
        }

        public void setAddress(Address address) {
            this.address = address;
        }

        public BigInteger getValue() {
            return value;
        }

        public void setValue(BigInteger value) {
            this.value = value;
        }
    }
}
