import React from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  ChartOptions,
  TooltipItem
} from 'chart.js';
import { Line } from 'react-chartjs-2';
import { cn } from '../../utils/classNames';

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend
);

export interface LineChartDataPoint {
  timestamp: Date;
  values: {
    [key: string]: number;
  };
  label: string;
}

interface LineChartSeries {
  key: string;
  label: string;
  color: string;
  yAxisID?: string;
}

interface SimpleLineChartProps {
  data: LineChartDataPoint[];
  series: LineChartSeries[];
  title?: string;
  className?: string;
  showLegend?: boolean;
  formatValue?: (value: number, key: string) => string;
}

export const SimpleLineChart: React.FC<SimpleLineChartProps> = ({
  data,
  series,
  title = "Line Chart",
  className,
  showLegend = true,
  formatValue = (value) => value.toLocaleString()
}) => {
  if (!data || data.length === 0) {
    return (
      <div className={cn("activity-card", className)}>
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-sm font-semibold text-white flex items-center">
            <span className="mr-2">📊</span>
            {title}
          </h3>
        </div>
        <div className="h-48 flex items-center justify-center text-gray-500">
          No data available
        </div>
      </div>
    );
  }

  // Prepare data for Chart.js
  const chartData = {
    labels: data.map(d => d.label),
    datasets: series.map((s, index) => ({
      label: s.label,
      data: data.map(d => d.values[s.key] || 0),
      borderColor: s.color,
      backgroundColor: s.color + '20',
      tension: 0.4,
      pointRadius: 0,
      pointHoverRadius: 6,
      borderWidth: 2,
      yAxisID: s.yAxisID || (index === 0 ? 'y' : 'y1')
    }))
  };

  const options: ChartOptions<'line'> = {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index',
      intersect: false,
    },
    plugins: {
      legend: {
        display: showLegend,
        position: 'top',
        align: 'end',
        labels: {
          color: '#9ca3af',
          usePointStyle: true,
          pointStyle: 'circle',
          padding: 20,
          font: {
            size: 12
          }
        }
      },
      tooltip: {
        backgroundColor: 'rgba(17, 24, 39, 0.95)',
        titleColor: '#f3f4f6',
        bodyColor: '#d1d5db',
        borderColor: '#374151',
        borderWidth: 1,
        padding: 12,
        displayColors: true,
        callbacks: {
          label: function(context: TooltipItem<'line'>) {
            const datasetIndex = context.datasetIndex;
            const seriesKey = series[datasetIndex]?.key;
            const value = context.parsed.y;
            const formattedValue = seriesKey ? formatValue(value, seriesKey) : value.toString();
            return `${context.dataset.label}: ${formattedValue}`;
          }
        }
      }
    },
    scales: {
      x: {
        grid: {
          display: false
        },
        ticks: {
          color: '#6b7280',
          font: {
            size: 11
          },
          maxRotation: 0,
          autoSkip: true,
          maxTicksLimit: 8
        }
      },
      y: {
        position: 'left',
        grid: {
          color: '#374151'
        },
        ticks: {
          color: '#6b7280',
          font: {
            size: 11
          },
          callback: function(value) {
            return formatValue(Number(value), 'cost');
          }
        }
      },
      y1: {
        position: 'right',
        grid: {
          display: false
        },
        ticks: {
          color: '#6b7280',
          font: {
            size: 11
          },
          callback: function(value) {
            return formatValue(Number(value), 'messages');
          }
        }
      }
    }
  };

  return (
    <div className={cn("activity-card", className)}>
      <div className="flex justify-between items-center mb-4">
        <h3 className="text-sm font-semibold text-white flex items-center">
          <span className="mr-2">📊</span>
          {title}
        </h3>
      </div>
      
      <div className="h-64">
        <Line data={chartData} options={options} />
      </div>
    </div>
  );
};