package org.aion.avm;

/**
 * This class performs the linear fee calculation for JCL classes.
 */
public class EnergyCalculator {

    /**
     * @param base base cost
     * @param linearValue linear cost
     * @return base + linearValue * RT_METHOD_FEE_FACTOR_LEVEL_2
     */
    public static long multiplyLinearValueByMethodFeeLevel2AndAddBase(int base, int linearValue) {
        return addAndCheckForOverflow(base, multiplyAndCheckForOverflow(linearValue, RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_2));
    }

    /**
     * @param base base cost
     * @param linearValue linear cost
     * @return base + linearValue * RT_METHOD_FEE_FACTOR_LEVEL_1
     */
    public static long multiplyLinearValueByMethodFeeLevel1AndAddBase(int base, int linearValue) {
        return addAndCheckForOverflow(base, multiplyAndCheckForOverflow(linearValue, RuntimeMethodFeeSchedule.RT_METHOD_FEE_FACTOR_LEVEL_1));
    }

    public static long multiply(int value1, int value2) {
        return multiplyAndCheckForOverflow(value1, value2);
    }

    private static long addAndCheckForOverflow(int value1, long value2) {
        return (long) value1 + value2;
    }

    private static long multiplyAndCheckForOverflow(int value1, int value2) {
        return (long) value1 * (long) value2;
    }
}
