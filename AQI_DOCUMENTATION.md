# Air Quality Index (AQI) Calculation Documentation

## Overview

This daemon calculates the Air Quality Index (AQI) based on PM2.5 and PM10 particulate matter concentrations using the United States Environmental Protection Agency (EPA) methodology.

## AQI Formula

The AQI is calculated using the following formula:

```
AQI = ((IHi - ILo) / (BPHi - BPLo)) × (Cp - BPLo) + ILo
```

Where:
- **IHi** = AQI value corresponding to BPHi
- **ILo** = AQI value corresponding to BPLo  
- **BPHi** = Concentration breakpoint that is greater than or equal to Cp
- **BPLo** = Concentration breakpoint that is less than or equal to Cp
- **Cp** = Truncated pollutant concentration

## Concentration Truncation

Before calculation, concentrations are truncated to one decimal place as per EPA guidelines. This is done by:
1. Multiplying by 10
2. Taking the floor value
3. Dividing by 10

Example: 35.49 µg/m³ becomes 35.4 µg/m³

## AQI Breakpoints

### PM2.5 Breakpoints (µg/m³, 24-hour average)

| Concentration Range | AQI Range | Category |
|-------------------|-----------|----------|
| 0.0 - 12.0 | 0 - 50 | Good |
| 12.1 - 35.4 | 51 - 100 | Moderate |
| 35.5 - 55.4 | 101 - 150 | Unhealthy for Sensitive Groups |
| 55.5 - 150.4 | 151 - 200 | Unhealthy |
| 150.5 - 250.4 | 201 - 300 | Very Unhealthy |
| 250.5 - 350.4 | 301 - 400 | Hazardous |
| 350.5 - 500.4 | 401 - 500 | Hazardous |

### PM10 Breakpoints (µg/m³, 24-hour average)

| Concentration Range | AQI Range | Category |
|-------------------|-----------|----------|
| 0 - 54 | 0 - 50 | Good |
| 55 - 154 | 51 - 100 | Moderate |
| 155 - 254 | 101 - 150 | Unhealthy for Sensitive Groups |
| 255 - 354 | 151 - 200 | Unhealthy |
| 355 - 424 | 201 - 300 | Very Unhealthy |
| 425 - 504 | 301 - 400 | Hazardous |
| 505 - 604 | 401 - 500 | Hazardous |

## Multiple Pollutant Handling

When multiple pollutants are measured:
1. Calculate AQI for each pollutant separately
2. Report the **highest** AQI value as the overall AQI
3. This ensures public health protection by reporting the worst air quality condition

In this implementation, we calculate AQI for both PM2.5 and PM10, then report the maximum value.

## Implementation Details

### Input Data Selection: Why pm02Standard Instead of pm02Compensated

The daemon uses the "standard" PM values (`pm02Standard` and `pm10Standard`) rather than the compensated values (`pm02Compensated`) for the following reasons:

1. **AQI Breakpoints are Based on FRM/FEM Standards**: The EPA AQI breakpoint concentrations are derived from National Ambient Air Quality Standards (NAAQS) and are based on Federal Reference Method (FRM) or Federal Equivalent Method (FEM) measurements. These reference methods produce what AirGradient reports as "standard" values.

2. **Avoiding Double Correction**: The `pm02Compensated` value has already been adjusted using the EPA 2021 correction formula for low-cost PM sensors. However, the AQI breakpoints themselves were not developed with this correction in mind. Applying AQI calculations to already-corrected values could result in an inaccurate AQI that doesn't align with EPA's intended public health thresholds.

3. **Regulatory Consistency**: For official AQI reporting, the EPA uses FRM/FEM data or data that has been shown to be equivalent to these reference methods. The "standard" values from the sensor are designed to approximate these reference measurements.

4. **EPA 2021 Correction Purpose**: The EPA 2021 correction formula (used to generate `pm02Compensated`) was primarily developed to improve the accuracy of low-cost sensors for research and supplemental monitoring purposes, not necessarily for direct AQI calculation.

If you need to use the compensated values for other analyses or comparisons, they may provide more accurate absolute PM2.5 concentrations, especially at higher humidity levels. However, for AQI calculation that aligns with EPA's public communication standards, the standard values are more appropriate.

### Example Calculation

For PM2.5 = 35.7 µg/m³:
1. Truncate to 35.7 (already one decimal)
2. Find breakpoints: BPLo = 35.5, BPHi = 55.4
3. Find AQI values: ILo = 101, IHi = 150
4. Calculate: AQI = ((150-101)/(55.4-35.5)) × (35.7-35.5) + 101
5. AQI = (49/19.9) × 0.2 + 101 = 101.5
6. Round to nearest integer: AQI = 102

## Health Implications

| AQI Range | Health Message |
|-----------|----------------|
| 0-50 | Air quality is satisfactory, and air pollution poses little or no risk |
| 51-100 | Air quality is acceptable. However, there may be a risk for some people who are unusually sensitive |
| 101-150 | Members of sensitive groups may experience health effects. The general public is less likely to be affected |
| 151-200 | Some members of the general public may experience health effects; members of sensitive groups may experience more serious health effects |
| 201-300 | Health alert: The risk of health effects is increased for everyone |
| 301+ | Health warning of emergency conditions: everyone is more likely to be affected |

## References

1. **EPA AQI Technical Assistance Document** (September 2018)  
   https://www.airnow.gov/sites/default/files/2020-05/aqi-technical-assistance-document-sept2018.pdf

2. **AirNow - AQI Basics**  
   https://www.airnow.gov/aqi/aqi-basics/

3. **EPA AQI Calculator**  
   https://www.airnow.gov/aqi/aqi-calculator/

4. **40 CFR Part 58 Appendix G** - Uniform Air Quality Index (AQI) and Daily Reporting  
   https://www.ecfr.gov/current/title-40/chapter-I/subchapter-C/part-58/appendix-Appendix%20G%20to%20Part%2058

## Data Source

The daemon processes data from AirGradient sensors, specifically using the PM2.5 and PM10 standard values which represent µg/m³ concentrations suitable for AQI calculations.