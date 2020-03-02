import React from 'react';
import PropTypes from 'prop-types';
import moment from 'moment';
import BatchStatusSummary from "./BatchStatusSummary.jsx";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {Link} from "react-router-dom";

class BatchListEntry extends React.Component {
    static propTypes = {
        entry: PropTypes.object.isRequired,
        entryWasDeleted: PropTypes.func.isRequired,
    };

    constructor(props) {
        super(props);

        this.state = {
            loading: false,
            lastError: "",
            summaryData: null,
        };

        this.deleteRecord = this.deleteRecord.bind(this);

    }

    nameToDisplay() {
        if(this.props.entry.nickName && this.props.entry.nickName!==""){
            return <div><p className="emphasis no-spacing">{this.props.entry.nickName}</p><p className="small no-spacing">{this.props.entry.bulkListId}</p></div>
        } else {
            return <p className="no-spacing">{this.props.entry.bulkListId}</p>
        }
    }

    setStatePromise(newState) {
        return new Promise((resolve,reject)=>this.setState(newState, ()=>resolve()));
    }

    async loadSummaryData() {
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/bulk/get?forId=" + this.props.entry.bulkListId);
        if(response.status===200){
            const content = await response.json();
            return this.setStatePromise({
                loading: false,
                lastError: null,
                summaryData: content
            });
        } else {
            const content = await response.text();
            try {
                const parsedErr = JSON.parse(content);
                return this.setStatePromise({
                    loading: false,
                    lastError: parsedErr.detail
                });
            } catch(e) {
                return this.setStatePromise({
                    loading: false,
                    lastError: content
                })
            }
        }
    }

    componentDidMount() {
        this.loadSummaryData();
    }

    async deleteRecord() {
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/bulk/delete?forId=" + this.props.entry.bulkListId, {method: "DELETE"});
        if(response.status===200){
            await response.body.cancel();
            await this.setStatePromise({loading:false});
            if(this.props.entryWasDeleted) this.props.entryWasDeleted(this.props.entry.bulkListId);
        } else {
            const content = await response.text();
            console.error("Could not delete list entry: ", content);
            await this.setStatePromise({loading: false, lastError: content});
        }
    }

    render() {
        return <li className="batch-master-container" key={this.props.entry.bulkListId}>
            <div className="batch-entry-cell icon"><Link to={"/batch/" + this.props.entry.bulkListId} style={{color: "inherit"}}><FontAwesomeIcon icon="layer-group"/></Link></div>
            <div className="batch-entry-cell baseline">{this.nameToDisplay()}</div>
            <div className="batch-entry-cell mini">{moment(this.props.entry.creationTime).format("ddd, MMM Do YYYY, h:mm:ss a")}</div>
            <div className="batch-entry-cell baseline">{this.state.summaryData ? <BatchStatusSummary batchStatus={this.state.summaryData}/> : <p className="error-text">{this.state.lastError}</p>} </div>
            <div className="batch-entry-cell icon"><FontAwesomeIcon icon="trash" className="clickable" onClick={this.deleteRecord}/></div>
        </li>
    }
}

export default BatchListEntry;