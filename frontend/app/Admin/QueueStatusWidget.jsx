import React from 'react';
import PropTypes from 'prop-types';
import {HorizontalBar} from 'react-chartjs-2';

class QueueStatusWidget extends React.Component {
    constructor(props) {
        super(props);

        this.state = {
            queueNames: [],
            queueSizes: [],
            loading: false,
            lastError: null,
            intervalTimer: null
        };

        this.refreshData = this.refreshData.bind(this);
    }

    static makeColourValues(count, offset){
        let values = [];
        for(let n=0;n<count;++n){
            let hue = (n/count)*360.0 + offset;
            values[n] = 'hsla(' + hue + ',75%,50%,0.6)'
        }
        return values;
    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()))
    }

    async refreshData() {
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/jobrunner/queuestats");

        if(response.status===200) {
            const content = await response.json();
            const update = {
                queueNames: Object.keys(content.queues),
                queueSizes: Object.keys(content.queues).map(k=>content.queues[k]),
                loading: false
            };
            return this.setStatePromise(update);
        } else {
            const errorText = await response.text();
            return this.setStatePromise({loading: false, lastError: errorText})
        }
    }

    componentDidMount() {
        this.setStatePromise({
            colourValues: [
                "rgba(19,0,215,0.79)","rgba(15,200,1,0.79)"
            ]
        }).then(()=> {
            this.refreshData().then(() => {
                const tmr = window.setInterval(this.refreshData, 3000);
                this.setState({intervalTimer: tmr});
            })
        });
    }

    componentWillUnmount() {
        if(this.state.intervalTimer) {
            window.clearInterval(this.state.intervalTimer);
        }
    }

    render() {
        return <div id="status-widget-holder" >
            <HorizontalBar
            data={{
                datasets: this.state.queueNames.map((name, idx)=> {
                    return {
                        label: name,
                        backgroundColor: this.state.colourValues[idx],
                        data: [this.state.queueSizes[idx]]
                    }
                })
            }}
            options={{
                scales: {
                    yAxes: [{
                        labels: ["Queue size"]
                    }],
                    xAxes: [{
                        ticks: {
                            min: 0
                        }
                    }]
                },
                maintainAspectRatio: false,
                height: "400px"
            }}
            height={300}
            />
        </div>
    }
}

export default QueueStatusWidget;