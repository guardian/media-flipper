import React from 'react';
import PropTypes from 'prop-types';
import ndjsonStream from "can-ndjson-stream";
import MenuBanner from "../MenuBanner.jsx";
import BatchEntry from "./BatchEntry.jsx";
import css from "../gridform.css";
import moment from 'moment';
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

class BatchEdit extends React.Component {
    constructor(props){
        super(props);

        this.state = {
            loading: false,
            lastError: null,
            batchId: "",
            batchCreated: "",
            batchNickname: "",
            pendingCount: 0,
            activeCount: 0,
            completedCount: 0,
            errorCount: 0,
            entries: [],
            currentReader: null,
            currentAbort: null
        };

    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()))
    }

    async loadExistingData(batchId) {
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/bulk/get?forId=" + batchId);
        if(response.status===200){
            const content = await response.json();
            return this.setStatePromise({
                loading: false,
                lastError: null,
                batchId: content.bulkListId,
                batchCreated: content.creationTime,
                pendingCount: content.pendingCount,
                activeCount: content.activeCount,
                completedCount: content.completedCount,
                errorCount: content.errorCount
            })
        } else {
            try {
                const jsonContent = await response.json();
                return this.setStatePromise({loading: false, lastError: jsonContent.detail});
            } catch (e) {
                console.error("could not get json error response from server: ", e);
                const bodyContent = await response.text();
                return this.setStatePromise({loading: false, lastError: bodyContent})
            }
        }
    }

    async loadBatchContent(batchId) {
        await this.setStatePromise({loading: true});

        const abortController = new AbortController();

        const response = await fetch("/api/bulk/content?forId=" + batchId, {signal: abortController.signal});
        const stream = await ndjsonStream(response.body);
        const reader = stream.getReader();

        await this.setStatePromise({currentReader: reader, currentAbort: abortController});

        function readNextChunk(reader) {
            reader.read().then(({done, value}) =>{
                if(value) {
                    console.log("Got ", value);
                    this.setState(oldState=>{
                        return {entries: oldState.entries.concat([value])}
                    })
                } else {
                    console.warn("Got no data");
                }
                if(done) {
                    this.setState({loading: false, lastError: null});
                } else {
                    readNextChunk(reader);
                }
            })
        }
        readNextChunk = readNextChunk.bind(this);
        readNextChunk(reader);
    }

    componentDidMount() {
        const batchId = this.props.match.params.batchId;
        if(batchId!=="new" && batchId!=="") {
            this.loadExistingData(batchId).then(()=>this.loadBatchContent(batchId))
        }
    }

    render() {
        return <div>
            <MenuBanner/>
            <h2>Edit Batch</h2>
            <div className="grid-form">
                <label className="grid-form-label" htmlFor="nickname">Friendly name</label>
                <input className="grid-form-control" id="nickname" value={this.state.batchNickname} onChange={evt=>this.setState({batchNickname: evt.target.value})}/>
                <label className="grid-form-label" htmlFor="created">Created at</label>
                <span id="created" className="grid-form-control">{moment(this.state.batchCreated).format("ddd, MMM Do YYYY, h:mm:ss a")}</span>
                <label className="grid-form-label" htmlFor="status">Status</label>
                <ul className="status-summary-container grid-form-control">
                    <li className="status-summary-entry"><FontAwesomeIcon icon="pause-circle" className="inline-icon" style={{color: "darkblue"}}/>{this.state.pendingCount} items pending</li>
                    <li className="status-summary-entry"><FontAwesomeIcon icon="play-circle" className="inline-icon" style={{color: "darkgreen"}}/>{this.state.activeCount} items active</li>
                    <li className="status-summary-entry"><FontAwesomeIcon icon="check-circle" className="inline-icon" style={{color: "darkgreen"}}/>{this.state.completedCount} items completed</li>
                    <li className="status-summary-entry"><FontAwesomeIcon icon="times-circle" className="inline-icon" style={{color: "darkred"}}/>{this.state.errorCount} items failed</li>
                </ul>
                <label className="grid-form-label" htmlFor="progress">Overall progress</label>
                <span id="progress" className="grid-form-control emphasis">{Math.ceil(100*(this.state.completedCount+this.state.errorCount)/(this.state.pendingCount+this.state.activeCount + this.state.completedCount + this.state.errorCount))} %</span>
            </div>
            <ul className="batch-list">
                {
                    this.state.entries.map(ent=><li key={ent.id}><BatchEntry entry={ent}/></li>)
                }
            </ul>
        </div>;
    }
}

export default BatchEdit;