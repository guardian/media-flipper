import React from 'react';
import PropTypes from 'prop-types';
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";

class JobStatusSummary extends React.Component {
    static propTypes = {
        className: PropTypes.string,
        filterClicked: PropTypes.func.isRequired,
        currentFilterName: PropTypes.string,
    };

    constructor(props) {
        super(props);

        this.state = {
            loading: false,
            lastError: null,
            jobStatus: {
                notqueued: 0,
                pending:0,
                started:0,
                completed:0,
                failed:0,
                aborted:0
            },
            timer: null
        };

        this.updateStats = this.updateStats.bind(this);
    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()))
    }

    async updateStats() {
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/job/summary/status");
        if(response.status===200) {
            const content = await response.json();
            return this.setStatePromise({
                loading: false,
                lastError: null,
                jobStatus: content.data
            })
        } else {
            const errorText = await response.text();
            console.error("could not update job status stats: ", errorText);
            return this.setStatePromise({
                loading: false,
                lastError: errorText
            });
        }
    }

    componentDidMount() {
        console.log("componentDidMount");
        this.updateStats().then(()=>{
            const timer = window.setInterval(this.updateStats, 1000);
            this.setState({timer: timer})
        })
    }

    componentWillUnmount() {
        console.log("componentDidUnMount");
        if(this.state.timer!==null) {
            window.clearInterval(this.state.timer);

        }
    }

    render() {
        const finalClassName = this.props.className ? "status-summary-container " + this.props.className : "status-summary-container";
        return <ul className={finalClassName}>
            <li className={"status-summary-entry clickable" + (this.props.currentFilterName==="notqueued" ? " selected" : "")} onClick={()=>this.props.filterClicked("notqueued")}>
                <FontAwesomeIcon icon="hand-paper" className="inline-icon" style={{color: "darkblue"}}/>{this.state.jobStatus.notqueued} items not queued
            </li>
            <li className={"status-summary-entry clickable" + (this.props.currentFilterName==="pending" ? " selected" : "")} onClick={()=>this.props.filterClicked("pending")}>
                <FontAwesomeIcon icon="pause-circle" className="inline-icon" style={{color: "darkblue"}}/>{this.state.jobStatus.pending} items pending
            </li>
            <li className={"status-summary-entry clickable" + (this.props.currentFilterName==="active" ? " selected" : "")} onClick={()=>this.props.filterClicked("active")}>
                <FontAwesomeIcon icon="play-circle" className="inline-icon" style={{color: "darkgreen"}}/>{this.state.jobStatus.started} items active
            </li>
            <li className={"status-summary-entry clickable" + (this.props.currentFilterName==="completed" ? " selected" : "")} onClick={()=>this.props.filterClicked("completed")}>
                <FontAwesomeIcon icon="check-circle" className="inline-icon" style={{color: "darkgreen"}}/>{this.state.jobStatus.completed} items completed
            </li>
            <li className={"status-summary-entry clickable" + (this.props.currentFilterName==="failed" ? " selected" : "")} onClick={()=>this.props.filterClicked("failed")}>
                <FontAwesomeIcon icon="times-circle" className="inline-icon" style={{color: "darkred"}}/>{this.state.jobStatus.failed} items failed
            </li>
            <li className={"status-summary-entry clickable" + (this.props.currentFilterName==="aborted" ? " selected" : "")} onClick={()=>this.props.filterClicked("aborted")}>
                <FontAwesomeIcon icon="eject" className="inline-icon" style={{color: "darkred"}}/>{this.state.jobStatus.aborted} items purged before running
            </li>
            <li className={"status-summary-entry clickable" + (this.props.currentFilterName==="lost" ? " selected" : "")} onClick={()=>this.props.filterClicked("lost")}>
                <FontAwesomeIcon icon="frown" className="inline-icon" style={{color:"darkred"}}/>{this.state.jobStatus.aborted} items lost
            </li>
        </ul>
    }
}

export default JobStatusSummary;
