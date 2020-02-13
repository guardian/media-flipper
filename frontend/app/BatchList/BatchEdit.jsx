import React from 'react';
import PropTypes from 'prop-types';
import ndjsonStream from "can-ndjson-stream";
import MenuBanner from "../MenuBanner.jsx";
import BatchEntry from "./BatchEntry.jsx";
import css from "../gridform.css";
import appcss from "../approot.css";

import moment from 'moment';
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import BatchStatusSummary from "./BatchStatusSummary.jsx";

class BatchEdit extends React.Component {
    constructor(props){
        super(props);

        this.state = {
            loading: false,
            lastError: null,
            batchId: "",
            batchCreated: "",
            batchNickname: "",
            templateId: "",
            pendingCount: 0,
            activeCount: 0,
            completedCount: 0,
            errorCount: 0,
            entries: [],
            currentReader: null,
            currentAbort: null,
            pageItemsLimit: 100,
            itemsInPage: 0,
        };

        this.triggerRemoveDotFiles = this.triggerRemoveDotFiles.bind(this);
    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()))
    }

    async triggerRemoveDotFiles() {
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/bulk/action/removeDotFiles?forId=" + this.state.batchId, {method: "POST"});
        if(response.status===200) {
            await response.body.cancel();
            await this.loadExistingData(this.state.batchId);
        } else {
            const bodyContent = await response.text();
            await this.setStatePromise({loading: false, lastError: bodyContent});
        }
    }

    async storeUpdatedInfo() {
        await this.setStatePromise({loading: true});
        const request = JSON.stringify({"nickName": this.state.batchNickname, "templateId": this.state.templateId});
        const response = await fetch("/api/bulk/update?forId=" + this.state.batchId, {
            body: request,
            method: "POST",
            headers: {"Content-Type": "application/json" },
        });

        if(response.status===200) {
            await response.body.cancel(); //don't actually care about the content
            console.log("update saved");
            return this.setStatePromise({loading: false, lastError: null});
        } else {
            const responseText = response.text();
            try {
                const content = JSON.parse(responseText);
                return this.setStatePromise({loading: false, lastError: content.detail});
            } catch(e) {
                return this.setStatePromise({loading: false, lastError: responseText});
            }
        }
    }

    componentDidUpdate(prevProps, prevState, snapshot) {
        if(this.state.batchNickname!==prevState.batchNickname || this.state.template!==prevState.template) {
            this.storeUpdatedInfo();
        }
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
                batchNickname: content.nickName,
                templateId: content.template,
                batchCreated: content.creationTime,
                pendingCount: content.pendingCount,
                activeCount: content.activeCount,
                completedCount: content.completedCount,
                errorCount: content.errorCount,
                removeFilesRunning: content.runningActions.includes("remove-system-files")
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

        function readNextChunk(reader, currentCount) {
            reader.read().then(({done, value}) =>{
                console.log(currentCount);
                if(value) {
                    //console.log("Got ", value);
                    this.setState(oldState=>{
                        return {entries: oldState.entries.concat([value]), itemsInPage: oldState.items +1}
                    }, ()=>{
                        if(done || currentCount>=this.state.pageItemsLimit-1) {
                            this.setState({loading: false, lastError: null});
                        } else {
                            readNextChunk(reader, currentCount+1);
                        }
                    })
                } else {
                    console.warn("Got no data");
                }
            })
        }
        readNextChunk = readNextChunk.bind(this);
        readNextChunk(reader,0);
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
                <BatchStatusSummary batchStatus={this.state} className="grid-form-control"/>
                <label className="grid-form-label" htmlFor="progress">Overall progress</label>
                <span id="progress" className="grid-form-control emphasis">{Math.ceil(100*(this.state.completedCount+this.state.errorCount)/(this.state.pendingCount+this.state.activeCount + this.state.completedCount + this.state.errorCount))} %</span>
                <label className="grid-form-label" htmlFor="actions">Actions</label>
                <span id="actions" className="grid-form-control">
                    <ul className="status-summary-container">
                        <li className={this.state.removeFilesRunning ? "status-summary-entry button disabled"  : "status-summary-entry button clickable"}
                            onClick={this.triggerRemoveDotFiles} style={{marginRight:"2em"}}>
                            <FontAwesomeIcon icon="minus-circle" style={{padding: "0.4em"}}/>Remove system files
                        </li>
                        <li className="status-summary-entry button clickable" onClick={this.triggerEnqueueItems} style={{marginRight:"2em"}}><FontAwesomeIcon icon="play-circle"  style={{padding: "0.4em"}}/>&nbsp;Start jobs running</li>
                    </ul>
                </span>
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